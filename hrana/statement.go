package hrana

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"yaagwadb"
	"yaagwadb/protocol"

	"github.com/pkg/errors"
	sqlparser "github.com/rqlite/sql"
)

const (
	columnPath  = "path"
	columnBody  = "body"
	columnSize  = "size"
	columnXattr = "xattr"
	// columnNotListed is the position of a column an INSERT does not list.
	columnNotListed = -1
	pathSplitter    = "/"
	// anonymousBinding is how a statement writes a parameter without a number or a name.
	anonymousBinding = "?"
	// numberedBindingFormat numbers an anonymous parameter, so that it keeps its number when moved to another statement.
	numberedBindingFormat = "?%d"
	// bodyPathAlias and bodySizeAlias name the columns added to a SELECT whose body the SQL front fetches.
	bodyPathAlias = "yaagwadb_body_path"
	bodySizeAlias = "yaagwadb_body_size"
	// base64Key names the base64 text of an extended attribute value that is not valid UTF-8.
	base64Key = "base64"
	// base64PaddingCharacter pads base64 text to a multiple of four characters.
	base64PaddingCharacter = "="
	// base64BytesPerGroup bytes are encoded as base64CharactersPerGroup characters.
	base64BytesPerGroup      = 3
	base64CharactersPerGroup = 4
	readFromStart            = 0
	readToEnd                = 0
	singleStatement          = 1
	firstStatementIndex      = 0
	singleColumn             = 1
	firstColumnIndex         = 0
	extraRowForLimitCheck    = 1
)

const (
	statementErrorFormatParse         = "%w: %v"
	statementErrorFormatCount         = "%w: one statement per execute, got %d"
	statementErrorFormatKind          = "%w: %T"
	statementErrorFormatTable         = "%w: only the objects table, not %q"
	statementErrorFormatBodyUse       = "%w: body can only be selected as a column"
	statementErrorFormatBodyShape     = "%w: body cannot be selected with DISTINCT, GROUP BY or a compound SELECT"
	statementErrorFormatLimit         = "%w: LIMIT %d is over the limit of %d rows"
	statementErrorFormatLimitValue    = "%w: LIMIT is not an integer"
	statementErrorFormatInsertColumns = "%w: INSERT takes the columns path and body"
	statementErrorFormatInsertSource  = "%w: INSERT takes VALUES"
	statementErrorFormatUpsert        = "%w: only ON CONFLICT DO UPDATE or DO NOTHING"
	statementErrorFormatExpression    = "%w: %s is not a value"
	statementErrorFormatBinding       = "%w: no value for parameter %s"
	statementErrorFormatValueType     = "%w: %s must be text"
	statementErrorFormatBase64        = "%w: body is not base64: %v"
	statementErrorFormatTooLarge      = "%w: %d bytes is over the limit of %d bytes"
	statementErrorFormatPath          = "%w: path %q"
	statementErrorFormatUpdateColumn  = "%w: only xattr can be updated, not %s"
	statementErrorFormatUpdateFrom    = "%w: UPDATE FROM"
	statementErrorFormatXattr         = "%w: xattr must be a JSON object of strings and {\"base64\": ...}: %v"
	statementErrorFormatTooManyRows   = "%w: more than %d rows match"
	statementErrorFormatDeleteClause  = "%w: DELETE with ORDER BY or LIMIT"
	statementErrorFormatReturning     = "%w: RETURNING"
)

const targetSelectFormat = `SELECT ` + columnPath + ` FROM ` + yaagwadb.ObjectsViewName + ` WHERE %s LIMIT %d`

const targetSelectAll = `SELECT ` + columnPath + ` FROM ` + yaagwadb.ObjectsViewName + ` LIMIT %d`

func invalidRequest() error {
	return fmt.Errorf(statementErrorFormatKind, errors.New(yaagwadb.InvalidArgument), nil)
}

func unsupportedRequest(kind string) error {
	return fmt.Errorf(statementErrorFormatParse, errors.New(RequestNotSupported), kind)
}

func (s *Server) execute(ctx context.Context, input *statement) (*statementResult, error) {
	if input.SQL == nil {
		return nil, fmt.Errorf(statementErrorFormatParse, errors.New(RequestNotSupported), requestTypeStoreSQL)
	}
	parsed, err := sqlparser.NewParser(strings.NewReader(*input.SQL)).ParseStatements()
	if err != nil {
		return nil, fmt.Errorf(statementErrorFormatParse, errors.New(yaagwadb.InvalidArgument), err)
	}
	if len(parsed) != singleStatement {
		return nil, fmt.Errorf(statementErrorFormatCount, errors.New(yaagwadb.InvalidArgument), len(parsed))
	}
	values, err := newArguments(input)
	if err != nil {
		return nil, err
	}
	statement := parsed[firstStatementIndex]
	// Numbering the anonymous parameters keeps each bound to its value when a clause moves to another statement
	if _, err := sqlparser.Walk(&bindingNumberer{}, statement); err != nil {
		return nil, err
	}
	switch typed := statement.(type) {
	case *sqlparser.SelectStatement:
		return s.selectStatement(ctx, *input.SQL, typed, values)
	case *sqlparser.InsertStatement:
		return s.insert(ctx, typed, values)
	case *sqlparser.UpdateStatement:
		return s.update(ctx, typed, values)
	case *sqlparser.DeleteStatement:
		return s.delete(ctx, typed, values)
	case *sqlparser.BeginStatement, *sqlparser.CommitStatement, *sqlparser.RollbackStatement,
		*sqlparser.SavepointStatement, *sqlparser.ReleaseStatement:
		return nil, errors.New(TransactionNotSupported)
	}
	return nil, fmt.Errorf(statementErrorFormatKind, errors.New(StatementNotSupported), statement)
}

type bindingNumberer struct{ count int }

func (v *bindingNumberer) Visit(node sqlparser.Node) (sqlparser.Visitor, sqlparser.Node, error) {
	if binding, ok := node.(*sqlparser.BindExpr); ok && binding.Name == anonymousBinding {
		v.count++
		binding.Name = fmt.Sprintf(numberedBindingFormat, v.count)
	}
	return v, node, nil
}

func (v *bindingNumberer) VisitEnd(node sqlparser.Node) (sqlparser.Node, error) { return node, nil }

func (a *arguments) resolve(expression sqlparser.Expr) (yaagwadb.QueryValue, error) {
	switch typed := expression.(type) {
	case *sqlparser.StringLit:
		return yaagwadb.QueryValue{Text: &typed.Value}, nil
	case *sqlparser.NumberLit:
		if integer, err := strconv.ParseInt(typed.Value, decimalBase, integerBitSize); err == nil {
			return yaagwadb.QueryValue{Integer: &integer}, nil
		}
		real, err := strconv.ParseFloat(typed.Value, integerBitSize)
		if err != nil {
			return yaagwadb.QueryValue{}, fmt.Errorf(statementErrorFormatParse, errors.New(yaagwadb.InvalidArgument), err)
		}
		return yaagwadb.QueryValue{Real: &real}, nil
	case *sqlparser.NullLit:
		return yaagwadb.QueryValue{}, nil
	case *sqlparser.BlobLit:
		return yaagwadb.QueryValue{}, fmt.Errorf(valueErrorFormatBlob, errors.New(TypeMismatch))
	case *sqlparser.BindExpr:
		return a.binding(typed.Name)
	}
	return yaagwadb.QueryValue{}, fmt.Errorf(statementErrorFormatExpression, errors.New(StatementNotSupported), expression)
}

func (a *arguments) binding(name string) (yaagwadb.QueryValue, error) {
	if number, err := strconv.Atoi(strings.TrimPrefix(name, anonymousBinding)); err == nil && strings.HasPrefix(name, anonymousBinding) {
		if number >= firstStatementPosition && number <= len(a.positional) {
			return a.positional[number-firstStatementPosition], nil
		}
	}
	if found, ok := a.named[name]; ok {
		return found, nil
	}
	return yaagwadb.QueryValue{}, fmt.Errorf(statementErrorFormatBinding, errors.New(yaagwadb.InvalidArgument), name)
}

func (a *arguments) text(expression sqlparser.Expr, what string) (string, error) {
	resolved, err := a.resolve(expression)
	if err != nil {
		return "", err
	}
	if resolved.Text == nil {
		return "", fmt.Errorf(statementErrorFormatValueType, errors.New(TypeMismatch), what)
	}
	return *resolved.Text, nil
}

func checkTable(name *sqlparser.Ident) error {
	if name == nil || !strings.EqualFold(name.Name, yaagwadb.ObjectsViewName) {
		table := ""
		if name != nil {
			table = name.Name
		}
		return fmt.Errorf(statementErrorFormatTable, errors.New(yaagwadb.InvalidArgument), table)
	}
	return nil
}

func isColumn(expression sqlparser.Expr, name string) bool {
	switch typed := expression.(type) {
	case *sqlparser.Ident:
		return strings.EqualFold(typed.Name, name)
	case *sqlparser.QualifiedRef:
		return typed.Column != nil && strings.EqualFold(typed.Column.Name, name)
	}
	return false
}

type columnCounter struct {
	name  string
	count int
}

func (v *columnCounter) Visit(node sqlparser.Node) (sqlparser.Visitor, sqlparser.Node, error) {
	if expression, ok := node.(sqlparser.Expr); ok && isColumn(expression, v.name) {
		v.count++
		// A qualified reference holds its column as an identifier, which is not counted again
		if _, qualified := expression.(*sqlparser.QualifiedRef); qualified {
			return nil, node, nil
		}
	}
	return v, node, nil
}

func (v *columnCounter) VisitEnd(node sqlparser.Node) (sqlparser.Node, error) { return node, nil }

func (s *Server) checkLimit(statement *sqlparser.SelectStatement, values *arguments) error {
	if statement.LimitExpr == nil {
		return nil
	}
	resolved, err := values.resolve(statement.LimitExpr)
	if err != nil {
		return err
	}
	if resolved.Integer == nil {
		return fmt.Errorf(statementErrorFormatLimitValue, errors.New(yaagwadb.InvalidArgument))
	}
	if *resolved.Integer > int64(s.config.QueryRowLimit) {
		return fmt.Errorf(statementErrorFormatLimit, errors.New(TooManyRows), *resolved.Integer, s.config.QueryRowLimit)
	}
	return nil
}

func (s *Server) query(ctx context.Context, statement string, values *arguments) (*yaagwadb.QueryResult, error) {
	return s.client.Query(ctx, statement, values.positional, values.named)
}

func (s *Server) selectStatement(ctx context.Context, original string, statement *sqlparser.SelectStatement, values *arguments) (*statementResult, error) {
	if err := s.checkLimit(statement, values); err != nil {
		return nil, err
	}
	counter := &columnCounter{name: columnBody}
	if _, err := sqlparser.Walk(counter, statement); err != nil {
		return nil, err
	}
	var bodyPositions []int
	var kept []*sqlparser.ResultColumn
	for position, resultColumn := range statement.Columns {
		if resultColumn.Expr != nil && isColumn(resultColumn.Expr, columnBody) {
			bodyPositions = append(bodyPositions, position)
			continue
		}
		kept = append(kept, resultColumn)
	}
	if counter.count != len(bodyPositions) {
		return nil, fmt.Errorf(statementErrorFormatBodyUse, errors.New(yaagwadb.InvalidArgument))
	}
	if len(bodyPositions) == 0 {
		result, err := s.query(ctx, original, values)
		if err != nil {
			return nil, err
		}
		return toStatementResult(result.Columns, result.Rows), nil
	}
	if statement.Distinct.IsValid() || statement.Compound != nil || len(statement.GroupByExprs) > 0 {
		return nil, fmt.Errorf(statementErrorFormatBodyShape, errors.New(yaagwadb.InvalidArgument))
	}
	statement.Columns = append(kept,
		&sqlparser.ResultColumn{Expr: &sqlparser.Ident{Name: columnPath}, Alias: &sqlparser.Ident{Name: bodyPathAlias}},
		&sqlparser.ResultColumn{Expr: &sqlparser.Ident{Name: columnSize}, Alias: &sqlparser.Ident{Name: bodySizeAlias}})
	result, err := s.query(ctx, statement.String(), values)
	if err != nil {
		return nil, err
	}
	return s.withBodies(ctx, result, bodyPositions)
}

func (s *Server) withBodies(ctx context.Context, result *yaagwadb.QueryResult, bodyPositions []int) (*statementResult, error) {
	keptCount := len(result.Columns) - len(bodyAliases)
	pathIndex, sizeIndex := keptCount, keptCount+1
	for _, row := range result.Rows {
		// Every row is checked before any file data is read, so a statement over the limit reads nothing
		if size := row[sizeIndex].Integer; size != nil && *size > s.config.FileDataSizeLimit {
			return nil, fmt.Errorf(statementErrorFormatTooLarge, errors.New(FileDataTooLarge), *size, s.config.FileDataSizeLimit)
		}
	}
	columns := insertAt(result.Columns[:keptCount], bodyPositions, columnBody)
	rows := make([][]yaagwadb.QueryValue, 0, len(result.Rows))
	for _, row := range result.Rows {
		var path string
		if row[pathIndex].Text != nil {
			path = *row[pathIndex].Text
		}
		content, err := s.readFile(ctx, path)
		if err != nil {
			return nil, err
		}
		encoded := base64.StdEncoding.EncodeToString(content)
		rows = append(rows, insertAt(row[:keptCount], bodyPositions, yaagwadb.QueryValue{Text: &encoded}))
	}
	return toStatementResult(columns, rows), nil
}

var bodyAliases = []string{bodyPathAlias, bodySizeAlias}

func insertAt[T any](items []T, positions []int, item T) []T {
	result := make([]T, 0, len(items)+len(positions))
	source := 0
	for position := 0; position < len(items)+len(positions); position++ {
		if containsPosition(positions, position) {
			result = append(result, item)
			continue
		}
		result = append(result, items[source])
		source++
	}
	return result
}

func containsPosition(positions []int, position int) bool {
	for _, candidate := range positions {
		if candidate == position {
			return true
		}
	}
	return false
}

func toStatementResult(columns []string, rows [][]yaagwadb.QueryValue) *statementResult {
	result := &statementResult{Columns: make([]column, 0, len(columns)), Rows: make([][]value, 0, len(rows))}
	for _, name := range columns {
		result.Columns = append(result.Columns, column{Name: name})
	}
	for _, row := range rows {
		converted := make([]value, 0, len(row))
		for _, item := range row {
			converted = append(converted, fromQueryValue(item))
		}
		result.Rows = append(result.Rows, converted)
	}
	return result
}

func splitPath(path string) ([]string, string, error) {
	names := strings.Split(path, pathSplitter)
	for _, name := range names {
		if err := yaagwadb.ValidateName(name); err != nil {
			return nil, "", fmt.Errorf(statementErrorFormatPath, err, path)
		}
	}
	last := len(names) - 1
	return names[:last], names[last], nil
}

func (s *Server) parentDirectory(ctx context.Context, directories []string) (uint64, error) {
	parent := uint64(yaagwadb.RootInodeNumber)
	for _, name := range directories {
		inode, err := s.client.Lookup(ctx, parent, name)
		if err != nil {
			return 0, err
		}
		parent = inode.InodeNumber
	}
	return parent, nil
}

func (s *Server) lookupPath(ctx context.Context, path string) (*yaagwadb.Inode, error) {
	directories, name, err := splitPath(path)
	if err != nil {
		return nil, err
	}
	parent, err := s.parentDirectory(ctx, directories)
	if err != nil {
		return nil, err
	}
	return s.client.Lookup(ctx, parent, name)
}

func (s *Server) readFile(ctx context.Context, path string) ([]byte, error) {
	inode, err := s.lookupPath(ctx, path)
	if err != nil {
		return nil, err
	}
	return s.client.Read(ctx, inode.InodeNumber, readFromStart, readToEnd)
}

func (s *Server) decodeBody(encoded string) ([]byte, error) {
	// The size is known from the length, so file data over the limit is refused before it is decoded
	padding := len(encoded) - len(strings.TrimRight(encoded, base64PaddingCharacter))
	if size := int64(len(encoded)*base64BytesPerGroup/base64CharactersPerGroup - padding); size > s.config.FileDataSizeLimit {
		return nil, fmt.Errorf(statementErrorFormatTooLarge, errors.New(FileDataTooLarge), size, s.config.FileDataSizeLimit)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf(statementErrorFormatBase64, errors.New(TypeMismatch), err)
	}
	return decoded, nil
}

func (s *Server) insert(ctx context.Context, statement *sqlparser.InsertStatement, values *arguments) (*statementResult, error) {
	if err := checkTable(statement.Table); err != nil {
		return nil, err
	}
	if statement.ReturningClause != nil {
		return nil, fmt.Errorf(statementErrorFormatReturning, errors.New(StatementNotSupported))
	}
	pathIndex, bodyIndex := columnNotListed, columnNotListed
	for index, name := range statement.Columns {
		switch {
		case strings.EqualFold(name.Name, columnPath):
			pathIndex = index
		case strings.EqualFold(name.Name, columnBody):
			bodyIndex = index
		default:
			return nil, fmt.Errorf(statementErrorFormatInsertColumns, errors.New(yaagwadb.InvalidArgument))
		}
	}
	if pathIndex == columnNotListed || bodyIndex == columnNotListed {
		return nil, fmt.Errorf(statementErrorFormatInsertColumns, errors.New(yaagwadb.InvalidArgument))
	}
	if statement.Select != nil || len(statement.ValueLists) == 0 {
		return nil, fmt.Errorf(statementErrorFormatInsertSource, errors.New(StatementNotSupported))
	}
	overwrite, skipExisting, err := upsertMode(statement.UpsertClauses)
	if err != nil {
		return nil, err
	}
	result := &statementResult{Columns: []column{}, Rows: [][]value{}}
	for _, list := range statement.ValueLists {
		if len(list.Exprs) != len(statement.Columns) {
			return nil, fmt.Errorf(statementErrorFormatInsertColumns, errors.New(yaagwadb.InvalidArgument))
		}
		path, err := values.text(list.Exprs[pathIndex], columnPath)
		if err != nil {
			return nil, err
		}
		encoded, err := values.text(list.Exprs[bodyIndex], columnBody)
		if err != nil {
			return nil, err
		}
		content, err := s.decodeBody(encoded)
		if err != nil {
			return nil, err
		}
		written, err := s.writeFile(ctx, path, content, overwrite, skipExisting)
		if err != nil {
			return nil, err
		}
		if written {
			result.AffectedRowCount++
		}
	}
	return result, nil
}

func upsertMode(clauses []*sqlparser.UpsertClause) (overwrite, skipExisting bool, err error) {
	if len(clauses) == 0 {
		return false, false, nil
	}
	if len(clauses) != singleStatement {
		return false, false, fmt.Errorf(statementErrorFormatUpsert, errors.New(StatementNotSupported))
	}
	clause := clauses[firstStatementIndex]
	switch {
	case clause.DoUpdate.IsValid() && clause.UpdateWhereExpr == nil:
		return true, false, nil
	case clause.DoNothing.IsValid():
		return false, true, nil
	}
	return false, false, fmt.Errorf(statementErrorFormatUpsert, errors.New(StatementNotSupported))
}

func (s *Server) writeFile(ctx context.Context, path string, content []byte, overwrite, skipExisting bool) (bool, error) {
	directories, name, err := splitPath(path)
	if err != nil {
		return false, err
	}
	parent, err := s.parentDirectory(ctx, directories)
	if err != nil {
		return false, err
	}
	_, err = s.client.Lookup(ctx, parent, name)
	switch {
	case err == nil && skipExisting:
		return false, nil
	case err == nil && !overwrite:
		return false, fmt.Errorf(statementErrorFormatPath, errors.New(yaagwadb.Exists), path)
	case err != nil && !yaagwadb.Has(err, yaagwadb.NotFound):
		return false, err
	}
	target := &protocol.WriteTarget{ParentInode: parent, Name: name, Mode: s.config.FileMode, Uid: s.config.UID, Gid: s.config.GID}
	if _, err := s.client.Write(ctx, target, bytes.NewReader(content)); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Server) targets(ctx context.Context, where sqlparser.Expr, values *arguments) ([]string, error) {
	statement := fmt.Sprintf(targetSelectAll, s.config.QueryRowLimit+extraRowForLimitCheck)
	if where != nil {
		statement = fmt.Sprintf(targetSelectFormat, where.String(), s.config.QueryRowLimit+extraRowForLimitCheck)
	}
	result, err := s.query(ctx, statement, values)
	if err != nil {
		return nil, err
	}
	if len(result.Rows) > s.config.QueryRowLimit {
		return nil, fmt.Errorf(statementErrorFormatTooManyRows, errors.New(TooManyRows), s.config.QueryRowLimit)
	}
	paths := make([]string, 0, len(result.Rows))
	for _, row := range result.Rows {
		if len(row) == singleColumn && row[firstColumnIndex].Text != nil {
			paths = append(paths, *row[firstColumnIndex].Text)
		}
	}
	return paths, nil
}

func (s *Server) update(ctx context.Context, statement *sqlparser.UpdateStatement, values *arguments) (*statementResult, error) {
	if statement.Table == nil {
		return nil, checkTable(nil)
	}
	if err := checkTable(statement.Table.Name); err != nil {
		return nil, err
	}
	if statement.Source != nil {
		return nil, fmt.Errorf(statementErrorFormatUpdateFrom, errors.New(StatementNotSupported))
	}
	if statement.ReturningClause != nil {
		return nil, fmt.Errorf(statementErrorFormatReturning, errors.New(StatementNotSupported))
	}
	var attributes map[string][]byte
	for _, assignment := range statement.Assignments {
		if len(assignment.Columns) != singleColumn || !strings.EqualFold(assignment.Columns[firstColumnIndex].Name, columnXattr) {
			return nil, fmt.Errorf(statementErrorFormatUpdateColumn, errors.New(yaagwadb.InvalidArgument), assignment.String())
		}
		text, err := values.text(assignment.Expr, columnXattr)
		if err != nil {
			return nil, err
		}
		if attributes, err = parseExtendedAttributes(text); err != nil {
			return nil, err
		}
	}
	paths, err := s.targets(ctx, statement.WhereExpr, values)
	if err != nil {
		return nil, err
	}
	result := &statementResult{Columns: []column{}, Rows: [][]value{}}
	for _, path := range paths {
		if err := s.replaceExtendedAttributes(ctx, path, attributes); err != nil {
			return nil, err
		}
		result.AffectedRowCount++
	}
	return result, nil
}

func parseExtendedAttributes(text string) (map[string][]byte, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, fmt.Errorf(statementErrorFormatXattr, errors.New(TypeMismatch), err)
	}
	attributes := make(map[string][]byte, len(raw))
	for name, encoded := range raw {
		var plain string
		if err := json.Unmarshal(encoded, &plain); err == nil {
			attributes[name] = []byte(plain)
			continue
		}
		var wrapped map[string]string
		if err := json.Unmarshal(encoded, &wrapped); err != nil {
			return nil, fmt.Errorf(statementErrorFormatXattr, errors.New(TypeMismatch), err)
		}
		decoded, err := base64.StdEncoding.DecodeString(wrapped[base64Key])
		if err != nil {
			return nil, fmt.Errorf(statementErrorFormatXattr, errors.New(TypeMismatch), err)
		}
		attributes[name] = decoded
	}
	return attributes, nil
}

func (s *Server) replaceExtendedAttributes(ctx context.Context, path string, attributes map[string][]byte) error {
	inode, err := s.lookupPath(ctx, path)
	if err != nil {
		return err
	}
	existing, err := s.client.Listxattr(ctx, inode.InodeNumber)
	if err != nil {
		return err
	}
	for _, name := range existing {
		if _, kept := attributes[name]; kept {
			continue
		}
		if err := s.client.Removexattr(ctx, inode.InodeNumber, name); err != nil {
			return err
		}
	}
	for name, content := range attributes {
		if err := s.client.Setxattr(ctx, inode.InodeNumber, name, content); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) delete(ctx context.Context, statement *sqlparser.DeleteStatement, values *arguments) (*statementResult, error) {
	if statement.Table == nil {
		return nil, checkTable(nil)
	}
	if err := checkTable(statement.Table.Name); err != nil {
		return nil, err
	}
	if len(statement.OrderingTerms) > 0 || statement.LimitExpr != nil {
		return nil, fmt.Errorf(statementErrorFormatDeleteClause, errors.New(StatementNotSupported))
	}
	if statement.ReturningClause != nil {
		return nil, fmt.Errorf(statementErrorFormatReturning, errors.New(StatementNotSupported))
	}
	paths, err := s.targets(ctx, statement.WhereExpr, values)
	if err != nil {
		return nil, err
	}
	result := &statementResult{Columns: []column{}, Rows: [][]value{}}
	for _, path := range paths {
		directories, name, err := splitPath(path)
		if err != nil {
			return nil, err
		}
		parent, err := s.parentDirectory(ctx, directories)
		if err != nil {
			return nil, err
		}
		if err := s.client.Unlink(ctx, parent, name); err != nil {
			return nil, err
		}
		result.AffectedRowCount++
	}
	return result, nil
}

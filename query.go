package yaagwadb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	sqlparser "github.com/rqlite/sql"

	"modernc.org/sqlite"
)

const (
	// ObjectsViewName is the only table read SQL can refer to.
	ObjectsViewName = "objects"
	// objectsTreeName is the common table expression inside the objects view that walks the directory tree.
	objectsTreeName = "yaagwadb_objects_tree"
	// base64FunctionName is the SQL function that encodes an extended attribute value that is not valid UTF-8.
	base64FunctionName = "yaagwadb_base64"
	// base64FunctionArgumentCount is the number of arguments of base64FunctionName.
	base64FunctionArgumentCount = 1
	// base64FunctionArgumentIndex is the position of the bytes to encode.
	base64FunctionArgumentIndex = 0
	// namedParameterPrefixes are the characters SQLite accepts in front of the name of a parameter.
	namedParameterPrefixes = ":@$"
)

// objectsViewSchema is the objects view. path is built from the directory tree: the names from the root joined by "/".
// xattr holds all extended attributes as one JSON object; a value that is not valid UTF-8 becomes {"base64": "..."}.
var objectsViewSchema = fmt.Sprintf(`CREATE VIEW IF NOT EXISTS objects AS
	WITH RECURSIVE %[6]s (inode_number, path) AS (
		SELECT inode_number, name FROM inode WHERE parent_inode_number = %[1]d AND inode_number != %[1]d
		UNION ALL
		SELECT child.inode_number, %[6]s.path || '/' || child.name
			FROM inode AS child JOIN %[6]s ON child.parent_inode_number = %[6]s.inode_number
	)
	SELECT
		%[6]s.path AS path,
		inode.size AS size,
		inode.modify_time AS mtime,
		file_data.state = %[2]d AS deleted,
		(SELECT json_group_object(attribute.name,
				CASE WHEN attribute.valid_utf8 THEN CAST(attribute.value AS TEXT)
					ELSE json_object('base64', %[5]s(attribute.value)) END)
			FROM extended_attribute AS attribute WHERE attribute.inode_number = inode.inode_number) AS xattr
	FROM %[6]s
		JOIN inode ON inode.inode_number = %[6]s.inode_number
		JOIN file_data ON file_data.file_data_id = inode.file_data_id
	WHERE inode.mode & %[3]d = %[4]d`, RootInodeNumber, FileDataLogicallyDeleted, ModeTypeMask, ModeRegular, base64FunctionName, objectsTreeName)

const (
	queryErrorFormatColumn         = "%w: column value of type %T"
	queryErrorFormatPrepare        = "%w: %v"
	queryErrorFormatStatementCount = "%w: read SQL must be one statement, got %d"
	queryErrorFormatNotSelect      = "%w: read SQL must be a SELECT statement"
	queryErrorFormatReservedName   = "%w: %q cannot name a common table expression"
)

const (
	singleStatement     = 1
	firstStatementIndex = 0
)

// QueryValue is one parameter or result value of read SQL. Exactly one field is set; none means NULL.
type QueryValue struct {
	Integer *int64
	Real    *float64
	Text    *string
}

// QueryResult is the result of read SQL.
type QueryResult struct {
	Columns []string
	Rows    [][]QueryValue
}

func encodeBase64Function(_ *sqlite.FunctionContext, arguments []driver.Value) (driver.Value, error) {
	switch value := arguments[base64FunctionArgumentIndex].(type) {
	case []byte:
		return base64.StdEncoding.EncodeToString(value), nil
	case string:
		return base64.StdEncoding.EncodeToString([]byte(value)), nil
	}
	return nil, nil
}

func installQueryAuthorizer(connection sqlite.ExecQuerierContext, _ string) error {
	registerer, ok := connection.(sqlite.AuthorizerRegisterer)
	if !ok {
		return errors.New(NotSupported)
	}
	return registerer.RegisterAuthorizer(authorizeQuery)
}

func authorizeQuery(action sqlite.AuthorizerActionCode, table, _, _, triggerOrView string) sqlite.AuthorizerReturnCode {
	// SQLite names the innermost view or common table expression a read happens in, so checkReadStatement
	// refuses a statement that defines a common table expression with either of these names
	switch action {
	case sqlite.AuthSelect, sqlite.AuthFunction, sqlite.AuthRecursive:
		return sqlite.AuthorizerOK
	case sqlite.AuthRead:
		if table == ObjectsViewName || triggerOrView == ObjectsViewName || triggerOrView == objectsTreeName {
			return sqlite.AuthorizerOK
		}
	}
	return sqlite.AuthorizerDeny
}

// reservedTableNames may not name a common table expression of read SQL; see authorizeQuery.
var reservedTableNames = []string{ObjectsViewName, objectsTreeName}

func checkReadStatement(statement string) error {
	statements, err := sqlparser.NewParser(strings.NewReader(statement)).ParseStatements()
	if err != nil {
		return fmt.Errorf(queryErrorFormatPrepare, errors.New(InvalidArgument), err)
	}
	if len(statements) != singleStatement {
		return fmt.Errorf(queryErrorFormatStatementCount, errors.New(InvalidArgument), len(statements))
	}
	if _, ok := statements[firstStatementIndex].(*sqlparser.SelectStatement); !ok {
		return fmt.Errorf(queryErrorFormatNotSelect, errors.New(InvalidArgument))
	}
	_, err = sqlparser.Walk(reservedNameVisitor{}, statements[firstStatementIndex])
	return err
}

type reservedNameVisitor struct{}

func (v reservedNameVisitor) Visit(node sqlparser.Node) (sqlparser.Visitor, sqlparser.Node, error) {
	if expression, ok := node.(*sqlparser.CTE); ok && expression.TableName != nil {
		for _, reserved := range reservedTableNames {
			if strings.EqualFold(expression.TableName.Name, reserved) {
				return nil, node, fmt.Errorf(queryErrorFormatReservedName, errors.New(InvalidArgument), expression.TableName.Name)
			}
		}
	}
	return v, node, nil
}

func (v reservedNameVisitor) VisitEnd(node sqlparser.Node) (sqlparser.Node, error) { return node, nil }

func openQueryDatabase(path string, busyTimeout time.Duration) (*sql.DB, error) {
	parameters := url.Values{}
	parameters.Add(sqliteParameterPragma, sqlitePragmaQueryOnly)
	query, err := sql.Open(queryDriverName, dataSourceName(path, busyTimeout, parameters))
	if err != nil {
		return nil, err
	}
	if err := query.Ping(); err != nil {
		query.Close()
		return nil, err
	}
	return query, nil
}

func queryArguments(parameters []QueryValue, named map[string]QueryValue) []any {
	arguments := make([]any, 0, len(parameters)+len(named))
	for _, parameter := range parameters {
		arguments = append(arguments, parameter.value())
	}
	for name, parameter := range named {
		// database/sql names a parameter without the prefix the statement writes it with
		arguments = append(arguments, sql.Named(strings.TrimLeft(name, namedParameterPrefixes), parameter.value()))
	}
	return arguments
}

func (v QueryValue) value() any {
	switch {
	case v.Integer != nil:
		return *v.Integer
	case v.Real != nil:
		return *v.Real
	case v.Text != nil:
		return *v.Text
	}
	return nil
}

func toQueryValue(value any) (QueryValue, error) {
	switch typed := value.(type) {
	case nil:
		return QueryValue{}, nil
	case int64:
		return QueryValue{Integer: &typed}, nil
	case float64:
		return QueryValue{Real: &typed}, nil
	case string:
		return QueryValue{Text: &typed}, nil
	case bool:
		integer := int64(0)
		if typed {
			integer = 1
		}
		return QueryValue{Integer: &integer}, nil
	case time.Time:
		text := typed.Format(time.RFC3339Nano)
		return QueryValue{Text: &text}, nil
	}
	return QueryValue{}, fmt.Errorf(queryErrorFormatColumn, errors.New(NotSupported), value)
}

func (s *Daemon) Query(ctx context.Context, statement string, parameters []QueryValue, named map[string]QueryValue) (*QueryResult, error) {
	if err := s.checkOpen(); err != nil {
		return nil, err
	}
	if err := checkReadStatement(statement); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, s.config.QueryTimeout)
	defer cancel()
	rows, err := s.query.QueryContext(ctx, statement, queryArguments(parameters, named)...)
	if err != nil {
		return nil, queryError(err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := &QueryResult{Columns: columns}
	for len(result.Rows) < s.config.QueryRowLimit && rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for index := range values {
			destinations[index] = &values[index]
		}
		if err := rows.Scan(destinations...); err != nil {
			return nil, err
		}
		row := make([]QueryValue, len(columns))
		for index, value := range values {
			if row[index], err = toQueryValue(value); err != nil {
				return nil, err
			}
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, queryError(err)
	}
	return result, nil
}

func queryError(err error) error {
	// The text of an SQLite error describes the statement, not the data directory, so it is passed on as InvalidArgument
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return err
	}
	var sqliteError *sqlite.Error
	if errors.As(err, &sqliteError) {
		return fmt.Errorf(queryErrorFormatPrepare, errors.New(InvalidArgument), sqliteError)
	}
	return err
}

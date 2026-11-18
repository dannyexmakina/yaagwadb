package hrana

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"yaagwadb"

	"github.com/pkg/errors"
	sqlparser "github.com/rqlite/sql"
)

const (
	columnMtime   = "mtime"
	columnDeleted = "deleted"
	// columnTextNotStarted means the text of the next result column has not started yet.
	columnTextNotStarted = -1
)

// declaredTypeInteger is the declared type of the inode columns that size and mtime come from.
const declaredTypeInteger = "INTEGER"

// objectsColumns are the columns of the objects table, in order. body is selected only by name.
var objectsColumns = []string{columnPath, columnSize, columnMtime, columnDeleted, columnXattr}

// objectsDeclaredTypes are the declared types SQLite reports for the columns of the objects view.
// The other columns are built by expressions in the view and have none.
var objectsDeclaredTypes = map[string]string{columnSize: declaredTypeInteger, columnMtime: declaredTypeInteger}

// columnTextEnds are the keywords that end the result columns of a SELECT.
var columnTextEnds = []sqlparser.Token{sqlparser.FROM, sqlparser.WHERE, sqlparser.GROUP, sqlparser.HAVING, sqlparser.WINDOW,
	sqlparser.ORDER, sqlparser.LIMIT, sqlparser.UNION, sqlparser.INTERSECT, sqlparser.EXCEPT, sqlparser.SEMI, sqlparser.EOF}

// describe tells the parameters and the result columns of a statement without running it.
func describe(sql *string) (*describeResult, error) {
	if sql == nil {
		return nil, fmt.Errorf(statementErrorFormatParse, errors.New(RequestNotSupported), requestTypeStoreSQL)
	}
	parsed, err := sqlparser.NewParser(strings.NewReader(*sql)).ParseStatements()
	if err != nil {
		return nil, fmt.Errorf(statementErrorFormatParse, errors.New(yaagwadb.InvalidArgument), err)
	}
	if len(parsed) != singleStatement {
		return nil, fmt.Errorf(statementErrorFormatCount, errors.New(yaagwadb.InvalidArgument), len(parsed))
	}
	statement := parsed[firstStatementIndex]
	collector := &parameterCollector{parameters: []describeParameter{}}
	if _, err := sqlparser.Walk(collector, statement); err != nil {
		return nil, err
	}
	result := &describeResult{Params: collector.parameters, Columns: []column{}}
	if selected, ok := statement.(*sqlparser.SelectStatement); ok {
		result.IsReadonly = true
		result.Columns = describeColumns(selected, resultColumnTexts(*sql), nil)
	}
	return result, nil
}

// parameterCollector lists the parameters of a statement by their number, as SQLite numbers them:
// ?NNN takes the number NNN, and a bare ? or a new name takes the next number after the largest so far.
type parameterCollector struct {
	parameters []describeParameter
}

func (v *parameterCollector) Visit(node sqlparser.Node) (sqlparser.Visitor, sqlparser.Node, error) {
	if binding, ok := node.(*sqlparser.BindExpr); ok {
		v.add(binding.Name)
	}
	return v, node, nil
}

func (v *parameterCollector) VisitEnd(node sqlparser.Node) (sqlparser.Node, error) { return node, nil }

func (v *parameterCollector) add(name string) {
	if name == anonymousBinding {
		v.parameters = append(v.parameters, describeParameter{})
		return
	}
	if number, err := strconv.Atoi(strings.TrimPrefix(name, anonymousBinding)); err == nil &&
		strings.HasPrefix(name, anonymousBinding) && number >= firstStatementPosition {
		for len(v.parameters) < number {
			v.parameters = append(v.parameters, describeParameter{})
		}
		v.parameters[number-firstStatementPosition].Name = &name
		return
	}
	for _, existing := range v.parameters {
		if existing.Name != nil && *existing.Name == name {
			return
		}
	}
	v.parameters = append(v.parameters, describeParameter{Name: &name})
}

// columnSource is a table that a SELECT reads from: the objects table, a common table expression or a subquery,
// under the name the statement gives it, with its columns in order.
type columnSource struct {
	name    string
	columns []column
}

// objectsSource is the objects table with the declared types of its columns.
func objectsSource(name string) columnSource {
	columns := make([]column, 0, len(objectsColumns))
	for _, objectsColumn := range objectsColumns {
		columns = append(columns, column{Name: objectsColumn, Decltype: declaredType(objectsColumn)})
	}
	return columnSource{name: name, columns: columns}
}

func declaredType(name string) *string {
	declared, ok := objectsDeclaredTypes[name]
	if !ok {
		return nil
	}
	return &declared
}

// describeColumns names the result columns of a SELECT and gives their declared types, as SQLite does:
// an alias names its column, a reference to a column of a table it reads takes that column's name and declared type,
// a scalar subquery takes the declared type of its column, and any other expression is named by its text as written
// and has no declared type. texts holds the text of each result column; without it, an expression is named
// by its normalized text. tables holds the common table expressions in scope, by name.
func describeColumns(statement *sqlparser.SelectStatement, texts []string, tables []columnSource) []column {
	if statement.WithClause != nil {
		for _, table := range statement.WithClause.CTEs {
			columns := describeColumns(table.Select, nil, tables)
			for index, name := range table.Columns {
				if index < len(columns) {
					columns[index].Name = name.Name
				}
			}
			tables = append([]columnSource{{name: table.TableName.Name, columns: columns}}, tables...)
		}
	}
	sources := sourcesOf(statement.Source, tables)
	columns := []column{}
	for index, resultColumn := range statement.Columns {
		if resultColumn.Star.IsValid() {
			for _, source := range sources {
				columns = append(columns, source.columns...)
			}
			continue
		}
		if reference, ok := resultColumn.Expr.(*sqlparser.QualifiedRef); ok && reference.Star.IsValid() {
			if source := findSource(sources, reference.Table); source != nil {
				columns = append(columns, source.columns...)
			}
			continue
		}
		described, named := describeExpression(resultColumn.Expr, sources, tables)
		if !named {
			described.Name = resultColumn.Expr.String()
			if len(texts) == len(statement.Columns) {
				described.Name = texts[index]
			}
		}
		if resultColumn.Alias != nil {
			described.Name = resultColumn.Alias.Name
		}
		columns = append(columns, described)
	}
	return columns
}

// describeExpression describes a result column that is an expression. A reference to a column, in parentheses
// or not, is named by the column; any other expression is not, and the caller names it by its text.
// A scalar subquery takes the declared type of its column, and any other expression that is not a reference has none.
func describeExpression(expression sqlparser.Expr, sources, tables []columnSource) (column, bool) {
	switch typed := expression.(type) {
	case *sqlparser.Ident:
		for _, source := range sources {
			if found := findColumn(source, typed.Name); found != nil {
				return *found, true
			}
		}
		return column{Name: typed.Name}, true
	case *sqlparser.QualifiedRef:
		if typed.Column == nil {
			return column{}, false
		}
		if source := findSource(sources, typed.Table); source != nil {
			if found := findColumn(*source, typed.Column.Name); found != nil {
				return *found, true
			}
		}
		return column{Name: typed.Column.Name}, true
	case *sqlparser.ParenExpr:
		subquery, ok := typed.X.(sqlparser.SelectExpr)
		if !ok {
			// A parenthesized expression is described as the expression inside
			return describeExpression(typed.X, sources, tables)
		}
		if columns := describeColumns(subquery.SelectStatement, nil, tables); len(columns) > 0 {
			return column{Decltype: columns[firstColumnIndex].Decltype}, false
		}
	}
	return column{}, false
}

// sourcesOf lists the tables a FROM clause reads, with their columns. A table it cannot describe has no columns.
func sourcesOf(source sqlparser.Source, tables []columnSource) []columnSource {
	switch typed := source.(type) {
	case *sqlparser.QualifiedTableName:
		name := typed.Name.Name
		if typed.Alias != nil {
			name = typed.Alias.Name
		}
		if table := findSource(tables, typed.Name); table != nil {
			return []columnSource{{name: name, columns: table.columns}}
		}
		if strings.EqualFold(typed.Name.Name, yaagwadb.ObjectsViewName) {
			return []columnSource{objectsSource(name)}
		}
		return []columnSource{{name: name}}
	case *sqlparser.ParenSource:
		if subquery, ok := typed.X.(*sqlparser.SelectStatement); ok {
			named := columnSource{columns: describeColumns(subquery, nil, tables)}
			if typed.Alias != nil {
				named.name = typed.Alias.Name
			}
			return []columnSource{named}
		}
		return sourcesOf(typed.X, tables)
	case *sqlparser.JoinClause:
		return append(sourcesOf(typed.X, tables), sourcesOf(typed.Y, tables)...)
	}
	return nil
}

func findSource(sources []columnSource, name *sqlparser.Ident) *columnSource {
	if name == nil {
		return nil
	}
	for index := range sources {
		if strings.EqualFold(sources[index].name, name.Name) {
			return &sources[index]
		}
	}
	return nil
}

// findColumn finds a column of a table by its name in any case, and returns it as the table names it.
func findColumn(source columnSource, name string) *column {
	for _, candidate := range source.columns {
		if strings.EqualFold(candidate.Name, name) {
			found := candidate
			return &found
		}
	}
	return nil
}

// resultColumnTexts cuts the text of each result column of the outermost SELECT out of a statement, as written.
// The scanner counts offsets in runes. Parentheses are tracked so that subqueries and calls are skipped.
func resultColumnTexts(sql string) []string {
	runes := []rune(sql)
	scanner := sqlparser.NewScanner(strings.NewReader(sql))
	var texts []string
	depth := 0
	start := columnTextNotStarted
	inColumns := false
	for {
		position, token, _ := scanner.Scan()
		if token == sqlparser.COMMENT {
			continue
		}
		topLevel := depth == 0
		switch token {
		case sqlparser.LP:
			depth++
		case sqlparser.RP:
			depth--
		}
		if !topLevel {
			continue
		}
		switch {
		case !inColumns:
			inColumns = token == sqlparser.SELECT
		case start == columnTextNotStarted:
			if token == sqlparser.DISTINCT || token == sqlparser.ALL {
				continue
			}
			start = position.Offset
		case token == sqlparser.COMMA:
			texts = append(texts, strings.TrimSpace(string(runes[start:position.Offset])))
			start = columnTextNotStarted
		case slices.Contains(columnTextEnds, token):
			end := min(position.Offset, len(runes))
			if token == sqlparser.EOF {
				end = len(runes)
			}
			return append(texts, strings.TrimSpace(string(runes[start:end])))
		}
		if token == sqlparser.EOF {
			return texts
		}
	}
}

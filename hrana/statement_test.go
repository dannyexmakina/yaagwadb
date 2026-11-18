package hrana

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"yaagwadb"
)

const (
	testPath              = "note.txt"
	testNestedDirectory   = "docs"
	testNestedPath        = "docs/inner.txt"
	testMissingParentPath = "missing/inner.txt"
	testContent           = "hello"
	testOtherContent      = "replaced"
	testTagValue          = "x"
	testXattrJSON         = `{"user.tag":"x","user.raw":{"base64":"//4A"}}`
	testInsert            = `INSERT INTO objects (path, body) VALUES (?, ?)`
	testUpsert            = `INSERT INTO objects (path, body) VALUES (?, ?) ON CONFLICT (path) DO UPDATE SET body = excluded.body`
	testInsertNothing     = `INSERT INTO objects (path, body) VALUES (?, ?) ON CONFLICT DO NOTHING`
	testInsertBlobLiteral = `INSERT INTO objects (path, body) VALUES ('a', x'00')`
	testSelectBody        = `SELECT body, path FROM objects WHERE path = ?`
	testSelectBodyInWhere = `SELECT path FROM objects WHERE body = 'x'`
	testSelectAll         = `SELECT * FROM objects ORDER BY path`
	testSelectByTag       = `SELECT path FROM objects WHERE xattr ->> 'user.tag' = :tag`
	testSelectLimit       = `SELECT path FROM objects LIMIT ?`
	testUpdateXattr       = `UPDATE objects SET xattr = ? WHERE path = ?`
	testUpdateSize        = `UPDATE objects SET size = 1`
	testDeleteWhere       = `DELETE FROM objects WHERE path = ?`
	testBegin             = `BEGIN`
	testOtherTable        = `SELECT * FROM inode`
	testTagParameter      = ":tag"
	testRawAttribute      = "user.raw"
	testTagAttribute      = "user.tag"
	testOverLimit         = testQueryRowLimit + 1
	testWithinLimit       = testQueryRowLimit
	testDirectoryMode     = 0o755
	testErrorCodeFormat   = "%s: want %s, got %v"
	testResultFormat      = "%s: %+v %v"
	testValueFormat       = "%s: got %q"
)

func text(input string) value { return textValue(input) }

func integerValue(input int) value {
	encoded, _ := json.Marshal(strconv.Itoa(input))
	return value{Type: valueTypeInteger, Value: encoded}
}

func encoded(content string) value {
	return text(base64.StdEncoding.EncodeToString([]byte(content)))
}

func run(t *testing.T, server *Server, sql string, arguments ...value) (*statementResult, error) {
	t.Helper()
	return server.execute(context.Background(), &statement{SQL: &sql, Args: arguments})
}

func mustRun(t *testing.T, server *Server, sql string, arguments ...value) *statementResult {
	t.Helper()
	result, err := run(t, server, sql, arguments...)
	if err != nil {
		t.Fatalf(testResultFormat, sql, result, err)
	}
	return result
}

func wantCode(t *testing.T, err error, code, what string) {
	t.Helper()
	got := errorCode(err)
	if got == nil || *got != code {
		t.Errorf(testErrorCodeFormat, what, code, err)
	}
}

func textOf(input value) string {
	var decoded string
	_ = json.Unmarshal(input.Value, &decoded)
	return decoded
}

func TestInsertAndSelectBody(t *testing.T) {
	server := startServer(t)
	if result := mustRun(t, server, testInsert, text(testPath), encoded(testContent)); result.AffectedRowCount != 1 {
		t.Fatalf(testResultFormat, testInsert, result, nil)
	}
	result := mustRun(t, server, testSelectBody, text(testPath))
	if len(result.Rows) != 1 || result.Columns[0].Name != columnBody || result.Columns[1].Name != columnPath ||
		textOf(result.Rows[0][0]) != base64.StdEncoding.EncodeToString([]byte(testContent)) || textOf(result.Rows[0][1]) != testPath {
		t.Fatalf(testResultFormat, testSelectBody, result, nil)
	}
	// SELECT * does not carry body
	result = mustRun(t, server, testSelectAll)
	for _, column := range result.Columns {
		if column.Name == columnBody {
			t.Fatalf(testResultFormat, testSelectAll, result, nil)
		}
	}
	_, err := run(t, server, testInsert, text(testPath), encoded(testOtherContent))
	wantCode(t, err, errorCodeExists, testInsert)
	mustRun(t, server, testUpsert, text(testPath), encoded(testOtherContent))
	result = mustRun(t, server, testSelectBody, text(testPath))
	if textOf(result.Rows[0][0]) != base64.StdEncoding.EncodeToString([]byte(testOtherContent)) {
		t.Fatalf(testValueFormat, testUpsert, textOf(result.Rows[0][0]))
	}
	if result := mustRun(t, server, testInsertNothing, text(testPath), encoded(testContent)); result.AffectedRowCount != 0 {
		t.Fatalf(testResultFormat, testInsertNothing, result, nil)
	}
}

func TestInsertIntoDirectory(t *testing.T) {
	server := startServer(t)
	if _, err := server.client.Mkdir(context.Background(), yaagwadb.RootInodeNumber, testNestedDirectory, testDirectoryMode, testOwnerID, testOwnerID); err != nil {
		t.Fatal(err)
	}
	mustRun(t, server, testInsert, text(testNestedPath), encoded(testContent))
	result := mustRun(t, server, testSelectBody, text(testNestedPath))
	if len(result.Rows) != 1 {
		t.Fatalf(testResultFormat, testSelectBody, result, nil)
	}
	_, err := run(t, server, testInsert, text(testMissingParentPath), encoded(testContent))
	wantCode(t, err, errorCodeNoEntry, testMissingParentPath)
}

func TestRefusals(t *testing.T) {
	server := startServer(t)
	mustRun(t, server, testInsert, text(testPath), encoded(testContent))
	tooLarge := strings.Repeat(testTagValue, testFileDataSizeLimit+1)
	_, err := run(t, server, testInsert, text(testPath), encoded(tooLarge))
	wantCode(t, err, errorCodeFileTooLarge, testInsert)
	_, err = run(t, server, testInsert, text(testPath), value{Type: valueTypeBlob})
	wantCode(t, err, errorCodeInvalidArgument, valueTypeBlob)
	_, err = run(t, server, testInsertBlobLiteral)
	wantCode(t, err, errorCodeInvalidArgument, testInsertBlobLiteral)
	_, err = run(t, server, testSelectBodyInWhere)
	wantCode(t, err, errorCodeInvalidArgument, testSelectBodyInWhere)
	_, err = run(t, server, testSelectLimit, integerValue(testOverLimit))
	wantCode(t, err, errorCodeInvalidArgument, testSelectLimit)
	mustRun(t, server, testSelectLimit, integerValue(testWithinLimit))
	_, err = run(t, server, testBegin)
	wantCode(t, err, errorCodeNotSupported, testBegin)
	_, err = run(t, server, testUpdateSize)
	wantCode(t, err, errorCodeInvalidArgument, testUpdateSize)
	_, err = run(t, server, testOtherTable)
	wantCode(t, err, errorCodeInvalidArgument, testOtherTable)
}

func TestSelectBodyOverLimit(t *testing.T) {
	server := startServer(t)
	mustRun(t, server, testInsert, text(testPath), encoded(testContent))
	server.config.FileDataSizeLimit = int64(len(testContent)) - 1
	_, err := run(t, server, testSelectBody, text(testPath))
	wantCode(t, err, errorCodeFileTooLarge, testSelectBody)
}

func TestUpdateExtendedAttributes(t *testing.T) {
	server := startServer(t)
	mustRun(t, server, testInsert, text(testPath), encoded(testContent))
	if result := mustRun(t, server, testUpdateXattr, text(testXattrJSON), text(testPath)); result.AffectedRowCount != 1 {
		t.Fatalf(testResultFormat, testUpdateXattr, result, nil)
	}
	tag := testTagValue
	result, err := server.execute(context.Background(), &statement{SQL: ptr(testSelectByTag),
		NamedArgs: []namedArgument{{Name: testTagParameter, Value: text(tag)}}})
	if err != nil || len(result.Rows) != 1 || textOf(result.Rows[0][0]) != testPath {
		t.Fatalf(testResultFormat, testSelectByTag, result, err)
	}
	inode, err := server.lookupPath(context.Background(), testPath)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := server.client.Getxattr(context.Background(), inode.InodeNumber, testRawAttribute)
	if err != nil || string(raw) != "\xff\xfe\x00" {
		t.Fatalf(testValueFormat, testRawAttribute, raw)
	}
	// Setting the column again replaces all attributes
	mustRun(t, server, testUpdateXattr, text(`{}`), text(testPath))
	if names, err := server.client.Listxattr(context.Background(), inode.InodeNumber); err != nil || len(names) != 0 {
		t.Fatalf(testResultFormat, testTagAttribute, names, err)
	}
}

func TestDelete(t *testing.T) {
	server := startServer(t)
	mustRun(t, server, testInsert, text(testPath), encoded(testContent))
	if result := mustRun(t, server, testDeleteWhere, text(testPath)); result.AffectedRowCount != 1 {
		t.Fatalf(testResultFormat, testDeleteWhere, result, nil)
	}
	if result := mustRun(t, server, testSelectAll); len(result.Rows) != 0 {
		t.Fatalf(testResultFormat, testSelectAll, result, nil)
	}
}

func ptr(input string) *string { return &input }

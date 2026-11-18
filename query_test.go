package yaagwadb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
)

const (
	testQueryAllObjects          = `SELECT path, size, deleted, xattr FROM objects ORDER BY path`
	testQueryByExtendedAttribute = `SELECT path FROM objects WHERE xattr ->> 'user.case' = ?`
	testQueryNamedParameter      = `SELECT path FROM objects WHERE path = :path`
	testQueryCount               = `SELECT COUNT(*), SUM(size) FROM objects`
	testQueryWithCTE             = `WITH named AS (SELECT path FROM objects) SELECT path FROM named`
	testNamedParameterName       = ":path"
	testNestedPath               = testDirectoryName + "/" + testInnerFileName
	testBinaryAttributeName      = "user.binary"
	testBase64Key                = "base64"
	testQueryRowLimit            = 2
	testQueryFileCount           = 3
	testQueryFileNameFormat      = "file-%d"
	// Failure message formats
	testQueryRowsFormat        = "rows %+v"
	testQueryWantDeniedFormat  = "%q: want InvalidArgument, got %v"
	testQueryExtendedAttribute = "xattr %q"
	testQueryRowCountFormat    = "got %d rows, want %d"
)

// testDeniedStatements may not run as read SQL: tables other than the objects view, the schema, PRAGMA, and writes.
var testDeniedStatements = []string{
	`SELECT * FROM inode`,
	`SELECT * FROM file_data`,
	`SELECT * FROM sqlite_master`,
	`SELECT * FROM objects JOIN extended_attribute`,
	`PRAGMA table_info(inode)`,
	`DELETE FROM inode`,
	`INSERT INTO meta VALUES ('x', 1)`,
	`ATTACH DATABASE 'other.db' AS other`,
	`CREATE TABLE x (y)`,
	`WITH objects AS (SELECT * FROM inode) SELECT * FROM objects`,
	`WITH Yaagwadb_Objects_Tree AS (SELECT * FROM inode) SELECT * FROM Yaagwadb_Objects_Tree`,
	`SELECT path FROM objects; SELECT path FROM objects`,
}

func text(value QueryValue) string {
	if value.Text == nil {
		return ""
	}
	return *value.Text
}

func integer(value QueryValue) int64 {
	if value.Integer == nil {
		return 0
	}
	return *value.Integer
}

func TestQueryObjectsView(t *testing.T) {
	daemon := openTestDaemon(t, testConfig(t.TempDir()))
	directory, err := daemon.Mkdir(RootInodeNumber, testDirectoryName, testDirectoryMode, testNoOwnerID, testNoOwnerID)
	if err != nil {
		t.Fatal(err)
	}
	nested := writeAndWait(t, daemon, directory.InodeNumber, testInnerFileName, []byte(testFileData))
	writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testSecondFileData))
	if err := daemon.SetExtendedAttribute(nested.InodeNumber, testExtendedAttributeName, []byte(testExtendedAttributeValue)); err != nil {
		t.Fatal(err)
	}
	binary := []byte{0xff, 0xfe, 0x00}
	if err := daemon.SetExtendedAttribute(nested.InodeNumber, testBinaryAttributeName, binary); err != nil {
		t.Fatal(err)
	}

	result, err := daemon.Query(context.Background(), testQueryAllObjects, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Directories are not rows; paths join the names from the root
	if len(result.Rows) != 2 || text(result.Rows[0][0]) != testNestedPath || text(result.Rows[1][0]) != testFileName ||
		integer(result.Rows[0][1]) != int64(len(testFileData)) || integer(result.Rows[0][2]) != 0 {
		t.Fatalf(testQueryRowsFormat, result.Rows)
	}
	var attributes map[string]any
	if err := json.Unmarshal([]byte(text(result.Rows[0][3])), &attributes); err != nil {
		t.Fatal(err)
	}
	encoded, _ := attributes[testBinaryAttributeName].(map[string]any)
	if attributes[testExtendedAttributeName] != testExtendedAttributeValue || encoded[testBase64Key] != base64.StdEncoding.EncodeToString(binary) {
		t.Fatalf(testQueryExtendedAttribute, text(result.Rows[0][3]))
	}

	value := testExtendedAttributeValue
	result, err = daemon.Query(context.Background(), testQueryByExtendedAttribute, []QueryValue{{Text: &value}}, nil)
	if err != nil || len(result.Rows) != 1 || text(result.Rows[0][0]) != testNestedPath {
		t.Fatalf(testQueryRowsFormat, result)
	}
	path := testFileName
	result, err = daemon.Query(context.Background(), testQueryNamedParameter, nil, map[string]QueryValue{testNamedParameterName: {Text: &path}})
	if err != nil || len(result.Rows) != 1 {
		t.Fatalf(testQueryRowsFormat, result)
	}
	result, err = daemon.Query(context.Background(), testQueryCount, nil, nil)
	if err != nil || integer(result.Rows[0][0]) != 2 || integer(result.Rows[0][1]) != int64(len(testFileData)+len(testSecondFileData)) {
		t.Fatalf(testQueryRowsFormat, result)
	}
	if result, err = daemon.Query(context.Background(), testQueryWithCTE, nil, nil); err != nil || len(result.Rows) != 2 {
		t.Fatalf(testQueryRowsFormat, result)
	}
}

func TestQueryDeniesOtherTables(t *testing.T) {
	daemon := openTestDaemon(t, testConfig(t.TempDir()))
	for _, statement := range testDeniedStatements {
		if _, err := daemon.Query(context.Background(), statement, nil, nil); !Has(err, InvalidArgument) {
			t.Errorf(testQueryWantDeniedFormat, statement, err)
		}
	}
}

func TestQueryRowLimit(t *testing.T) {
	config := testConfig(t.TempDir())
	config.QueryRowLimit = testQueryRowLimit
	daemon := openTestDaemon(t, config)
	for index := range testQueryFileCount {
		writeAndWait(t, daemon, RootInodeNumber, fmt.Sprintf(testQueryFileNameFormat, index), []byte(testFileData))
	}
	result, err := daemon.Query(context.Background(), testQueryAllObjects, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != testQueryRowLimit {
		t.Fatalf(testQueryRowCountFormat, len(result.Rows), testQueryRowLimit)
	}
}

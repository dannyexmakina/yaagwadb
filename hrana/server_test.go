package hrana

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"yaagwadb"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	testSocketDirectoryPrefix = "hrana"
	testSocketFileName        = "daemon.sock"
	testDefaultTemporary      = ""
	testFileDataSizeLimit     = 16
	testQueryRowLimit         = 3
	testFileMode              = 0o644
	testOwnerID               = 0
	testStatusFormat          = "status %d"
	testResultsFormat         = "results %+v"
	testSelectPaths           = `SELECT path FROM objects`
	testInvalidBody           = "{"
	testInvalidProtobuf       = "\xff"
	testDescribeFormat        = "%s: described %+v, ran %+v"
)

func startServer(t *testing.T) *Server {
	t.Helper()
	config := yaagwadb.DefaultConfig()
	config.DataDirectory = t.TempDir()
	config.DisableScheduler = true
	config.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	daemon, err := yaagwadb.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	// A Unix socket path has a short length limit, so the socket goes in a short temporary directory
	directory, err := os.MkdirTemp(testDefaultTemporary, testSocketDirectoryPrefix)
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(directory, testSocketFileName)
	server, err := yaagwadb.ListenGRPC(daemon, socket)
	if err != nil {
		t.Fatal(err)
	}
	client, err := yaagwadb.Dial(socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		client.Close()
		server.Stop()
		daemon.Close()
		os.RemoveAll(directory)
	})
	return NewServer(client, Config{FileDataSizeLimit: testFileDataSizeLimit, QueryRowLimit: testQueryRowLimit,
		FileMode: testFileMode, UID: testOwnerID, GID: testOwnerID, Logger: config.Logger})
}

func post(handler http.Handler, path string, body []byte) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body)))
	return recorder
}

func postPipeline(t *testing.T, handler http.Handler, request pipelineRequest) (int, pipelineResponse) {
	t.Helper()
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	recorder := post(handler, jsonVersionPath+pipelinePathSuffix, body)
	var response pipelineResponse
	if recorder.Code == http.StatusOK {
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
	}
	return recorder.Code, response
}

func TestVersion(t *testing.T) {
	// Neither the version check nor a body that cannot be read reaches the daemon
	handler := NewServer(nil, Config{}).Handler()
	for _, path := range []string{jsonVersionPath, protobufVersionPath} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf(testStatusFormat, recorder.Code)
		}
	}
}

func TestPipeline(t *testing.T) {
	handler := startServer(t).Handler()
	selectPaths := testSelectPaths
	begin := testBegin
	stepFailed := uint32(1)
	code, response := postPipeline(t, handler, pipelineRequest{Requests: []streamRequest{
		{Type: requestTypeExecute, Stmt: &statement{SQL: &selectPaths}},
		{Type: requestTypeBatch, Batch: &batch{Steps: []batchStep{
			{Stmt: statement{SQL: &selectPaths}},
			{Stmt: statement{SQL: &begin}},
			{Condition: &batchCondition{Type: conditionTypeError, Step: &stepFailed}, Stmt: statement{SQL: &selectPaths}},
		}}},
		{Type: requestTypeDescribe, SQL: &selectPaths},
		{Type: requestTypeGetAutocommit},
		{Type: requestTypeSequence, SQL: &selectPaths},
		{Type: requestTypeClose},
	}})
	if code != http.StatusOK || len(response.Results) != 6 || response.Baton != nil {
		t.Fatalf(testResultsFormat, response)
	}
	results := response.Results
	if results[0].Type != resultTypeOK || results[0].Response.Type != requestTypeExecute ||
		results[1].Type != resultTypeOK || results[1].Response.Type != requestTypeBatch ||
		results[2].Type != resultTypeOK || results[2].Response.Type != requestTypeDescribe ||
		results[3].Type != resultTypeOK || results[3].Response.IsAutocommit == nil || !*results[3].Response.IsAutocommit ||
		results[4].Type != resultTypeError || *results[4].Error.Code != errorCodeNotSupported ||
		results[5].Type != resultTypeOK {
		t.Fatalf(testResultsFormat, results)
	}
}

func TestBatch(t *testing.T) {
	server := startServer(t)
	selectPaths := testSelectPaths
	begin := testBegin
	first, second := uint32(0), uint32(1)
	result := server.runBatch(t.Context(), &batch{Steps: []batchStep{
		{Stmt: statement{SQL: &selectPaths}},
		{Stmt: statement{SQL: &begin}},
		{Condition: &batchCondition{Type: conditionTypeOK, Step: &second}, Stmt: statement{SQL: &selectPaths}},
		{Condition: &batchCondition{Type: conditionTypeAnd, Conds: []batchCondition{
			{Type: conditionTypeOK, Step: &first}, {Type: conditionTypeError, Step: &second}}}, Stmt: statement{SQL: &selectPaths}},
		{Condition: &batchCondition{Type: testUnknownConditionType}, Stmt: statement{SQL: &selectPaths}},
	}})
	if result.StepResults[0] == nil || result.StepErrors[1] == nil || *result.StepErrors[1].Code != errorCodeNotSupported ||
		result.StepResults[2] != nil || result.StepErrors[2] != nil ||
		result.StepResults[3] == nil || result.StepErrors[4] == nil {
		t.Fatalf(testResultsFormat, result)
	}
}

func TestCursor(t *testing.T) {
	handler := startServer(t).Handler()
	insert, selectPaths := testInsert, testSelectPaths
	body, err := json.Marshal(cursorRequest{Batch: &batch{Steps: []batchStep{
		{Stmt: statement{SQL: &insert, Args: []value{text(testPath), encoded(testContent)}}},
		{Stmt: statement{SQL: &selectPaths}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	recorder := post(handler, jsonVersionPath+cursorPathSuffix, body)
	if recorder.Code != http.StatusOK {
		t.Fatalf(testStatusFormat, recorder.Code)
	}
	var types []string
	scanner := bufio.NewScanner(recorder.Body)
	for scanner.Scan() {
		var entry struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatal(err)
		}
		types = append(types, entry.Type)
	}
	// The first line is the cursor response, which has no type
	want := []string{"", entryTypeStepBegin, entryTypeStepEnd, entryTypeStepBegin, entryTypeRow, entryTypeStepEnd}
	if len(types) != len(want) {
		t.Fatalf(testResultsFormat, types)
	}
	for index := range want {
		if types[index] != want[index] {
			t.Fatalf(testResultsFormat, types)
		}
	}
}

// describeQueries are SELECTs whose described columns are compared with those of running them.
var describeQueries = []string{
	`SELECT * FROM objects`,
	`SELECT SIZE, o.Path, length( path /* c */ ), mtime AS m, path || '/' FROM objects AS o`,
	`SELECT (size), (SELECT max(size) FROM objects), count(*) FROM objects`,
	`WITH t (x) AS (SELECT size FROM objects) SELECT x, q.* FROM t, (SELECT path AS p FROM objects) AS q`,
}

// TestDescribeMatchesDaemon checks that describe names the columns as the daemon's SQLite does when it runs the statement.
func TestDescribeMatchesDaemon(t *testing.T) {
	server := startServer(t)
	for _, query := range describeQueries {
		described, err := describe(&query)
		if err != nil {
			t.Fatal(err)
		}
		executed := mustRun(t, server, query)
		if len(described.Columns) != len(executed.Columns) {
			t.Fatalf(testDescribeFormat, query, described.Columns, executed.Columns)
		}
		for index := range executed.Columns {
			if described.Columns[index].Name != executed.Columns[index].Name {
				t.Fatalf(testDescribeFormat, query, described.Columns, executed.Columns)
			}
		}
	}
}

func TestProtobufPipeline(t *testing.T) {
	handler := startServer(t).Handler()
	execute := appendMessageField(nil, fieldExecuteStatement, appendStringField(nil, fieldStatementSQL, testSelectPaths))
	request := appendMessageField(nil, testExecuteField, execute)
	recorder := post(handler, protobufVersionPath+pipelinePathSuffix, appendMessageField(nil, fieldPipelineRequests, request))
	if recorder.Code != http.StatusOK || recorder.Header().Get(headerContentType) != protobufContentType {
		t.Fatalf(testStatusFormat, recorder.Code)
	}
	results := fieldsOf(t, recorder.Body.Bytes())[fieldPipelineResults]
	if len(results) != 1 || len(fieldsOf(t, results[0])[fieldResultOK]) != 1 {
		t.Fatalf(testResultsFormat, results)
	}
}

func TestRejectsInvalidBodies(t *testing.T) {
	handler := NewServer(nil, Config{}).Handler()
	for path, body := range map[string]string{
		jsonVersionPath + pipelinePathSuffix:     testInvalidBody,
		jsonVersionPath + cursorPathSuffix:       testInvalidBody,
		protobufVersionPath + pipelinePathSuffix: testInvalidProtobuf,
		protobufVersionPath + cursorPathSuffix:   testInvalidProtobuf,
	} {
		if recorder := post(handler, path, []byte(body)); recorder.Code != http.StatusBadRequest {
			t.Fatalf(testStatusFormat, recorder.Code)
		}
	}
}

// fieldsOf splits a Protobuf message into the contents of its length-delimited fields, by number.
func fieldsOf(t *testing.T, message []byte) map[protowire.Number][][]byte {
	t.Helper()
	fields := map[protowire.Number][][]byte{}
	err := eachField(message, func(number protowire.Number, kind protowire.Type, field []byte) error {
		if kind != protowire.BytesType {
			return nil
		}
		content, err := bytesField(number, kind, field)
		fields[number] = append(fields[number], content)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return fields
}

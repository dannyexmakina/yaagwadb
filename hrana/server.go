package hrana

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"yaagwadb"

	"github.com/pkg/errors"
)

const (
	TypeMismatch            = "hrana: type mismatch"
	FileDataTooLarge        = "hrana: file data too large"
	TransactionNotSupported = "hrana: transactions are not supported"
	RequestNotSupported     = "hrana: request type not supported"
	StatementNotSupported   = "hrana: statement not supported"
	TooManyRows             = "hrana: too many rows"
)

const (
	errorCodeNoEntry         = "ENOENT"
	errorCodeExists          = "EEXIST"
	errorCodeInvalidArgument = "EINVAL"
	errorCodeFileTooLarge    = "EFBIG"
	errorCodeBusy            = "EBUSY"
	errorCodeInputOutput     = "EIO"
	errorCodeNotSupported    = "ENOTSUP"
	errorCodeIsDirectory     = "EISDIR"
	errorCodeNotDirectory    = "ENOTDIR"
	errorCodeNotEmpty        = "ENOTEMPTY"
	errorCodeNoSpace         = "ENOSPC"
)

const (
	logRequestFailed = "hrana request"
	logKeyError      = "error"
	logKeyStack      = "stack"
)

const (
	conditionErrorFormat = "%w: batch condition %q"
	// jsonLineSeparator ends each JSON structure of a cursor response.
	jsonLineSeparator = '\n'
)

// errorCodes gives the errno-like code of each error constant, after the daemon's errno policy.
var errorCodes = []struct {
	message string
	code    string
}{
	{FileDataTooLarge, errorCodeFileTooLarge},
	{TypeMismatch, errorCodeInvalidArgument},
	{TransactionNotSupported, errorCodeNotSupported},
	{RequestNotSupported, errorCodeNotSupported},
	{StatementNotSupported, errorCodeNotSupported},
	{TooManyRows, errorCodeInvalidArgument},
	{yaagwadb.NotFound, errorCodeNoEntry},
	{yaagwadb.Exists, errorCodeExists},
	{yaagwadb.InvalidArgument, errorCodeInvalidArgument},
	{yaagwadb.Locked, errorCodeBusy},
	{yaagwadb.Corrupted, errorCodeInputOutput},
	{yaagwadb.IsDirectory, errorCodeIsDirectory},
	{yaagwadb.NotDirectory, errorCodeNotDirectory},
	{yaagwadb.NotEmpty, errorCodeNotEmpty},
	{yaagwadb.NotSupported, errorCodeNotSupported},
	{yaagwadb.NoSpace, errorCodeNoSpace},
	{yaagwadb.Closed, errorCodeInputOutput},
}

// Config is the configuration of the SQL front.
type Config struct {
	// FileDataSizeLimit is the largest file data, after base64 decoding, that INSERT accepts and SELECT returns.
	FileDataSizeLimit int64
	// QueryRowLimit is the largest LIMIT a SELECT may give. It matches the daemon's limit on rows of read SQL.
	QueryRowLimit int
	// FileMode, UID and GID are given to the files INSERT creates.
	FileMode uint32
	UID, GID uint32
	Logger   *slog.Logger
}

// Server serves Hrana 3 over HTTP, in JSON and in Protobuf, and passes each statement to the daemon over gRPC.
type Server struct {
	client *yaagwadb.Client
	config Config
}

func NewServer(client *yaagwadb.Client, config Config) *Server {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	return &Server{client: client, config: config}
}

// encoding reads the request bodies and writes the response bodies of one of the two encodings of Hrana 3.
type encoding struct {
	contentType   string
	readPipeline  func([]byte) (*pipelineRequest, error)
	writePipeline func(*pipelineResponse) ([]byte, error)
	readCursor    func([]byte) (*cursorRequest, error)
	// writeCursor and writeEntry write one structure of a cursor response, with what separates it from the next.
	writeCursor func(*cursorResponse) ([]byte, error)
	writeEntry  func(any) ([]byte, error)
}

var jsonEncoding = encoding{
	contentType:   jsonContentType,
	readPipeline:  readJSON[pipelineRequest],
	writePipeline: writeJSON[*pipelineResponse],
	readCursor:    readJSON[cursorRequest],
	writeCursor:   writeJSONLine[*cursorResponse],
	writeEntry:    writeJSONLine[any],
}

var protobufEncoding = encoding{
	contentType:   protobufContentType,
	readPipeline:  readProtobufPipeline,
	writePipeline: writeProtobufPipeline,
	readCursor:    readProtobufCursor,
	writeCursor:   writeProtobufCursor,
	writeEntry:    writeProtobufEntry,
}

func readJSON[T any](body []byte) (*T, error) {
	var decoded T
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, err
	}
	return &decoded, nil
}

func writeJSON[T any](item T) ([]byte, error) { return json.Marshal(item) }

func writeJSONLine[T any](item T) ([]byte, error) {
	encoded, err := json.Marshal(item)
	return append(encoded, jsonLineSeparator), err
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	for version, body := range map[string]encoding{jsonVersionPath: jsonEncoding, protobufVersionPath: protobufEncoding} {
		mux.HandleFunc(http.MethodGet+" "+version, func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
		})
		mux.HandleFunc(http.MethodPost+" "+version+pipelinePathSuffix, s.pipeline(body))
		mux.HandleFunc(http.MethodPost+" "+version+cursorPathSuffix, s.cursor(body))
	}
	return mux
}

func (s *Server) pipeline(body encoding) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		content, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		input, err := body.readPipeline(content)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		baton := emptyBaton
		response := pipelineResponse{Baton: &baton, Results: make([]streamResult, 0, len(input.Requests))}
		for _, streamRequest := range input.Requests {
			result, closed := s.handle(request.Context(), streamRequest)
			response.Results = append(response.Results, result)
			if closed {
				// No baton tells the client that the stream is closed
				response.Baton = nil
			}
		}
		output, err := body.writePipeline(&response)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set(headerContentType, body.contentType)
		_, _ = writer.Write(output)
	}
}

// cursor runs a batch and writes its results as cursor entries. The whole batch runs before the first entry is written.
func (s *Server) cursor(body encoding) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		content, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		input, err := body.readCursor(content)
		if err == nil && input.Batch == nil {
			err = invalidRequest()
		}
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		result := s.runBatch(request.Context(), input.Batch)
		baton := emptyBaton
		head, err := body.writeCursor(&cursorResponse{Baton: &baton})
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set(headerContentType, body.contentType)
		if _, err := writer.Write(head); err != nil {
			return
		}
		for _, entry := range cursorEntries(result) {
			encoded, err := body.writeEntry(entry)
			if err != nil {
				return
			}
			if _, err := writer.Write(encoded); err != nil {
				return
			}
		}
	}
}

// handle runs one request of a pipeline. It also reports whether the request closed the stream.
func (s *Server) handle(ctx context.Context, request streamRequest) (streamResult, bool) {
	switch request.Type {
	case requestTypeExecute:
		if request.Stmt == nil {
			return s.errorResult(invalidRequest()), false
		}
		result, err := s.executeStatement(ctx, request.Stmt)
		if err != nil {
			return s.errorResult(err), false
		}
		return okResult(&streamResponse{Type: requestTypeExecute, Result: result}), false
	case requestTypeBatch:
		if request.Batch == nil {
			return s.errorResult(invalidRequest()), false
		}
		return okResult(&streamResponse{Type: requestTypeBatch, Result: s.runBatch(ctx, request.Batch)}), false
	case requestTypeDescribe:
		result, err := describe(request.SQL)
		if err != nil {
			return s.errorResult(err), false
		}
		return okResult(&streamResponse{Type: requestTypeDescribe, Result: result}), false
	case requestTypeGetAutocommit:
		autocommit := alwaysAutocommit
		return okResult(&streamResponse{Type: requestTypeGetAutocommit, IsAutocommit: &autocommit}), false
	case requestTypeClose:
		return okResult(&streamResponse{Type: requestTypeClose}), true
	}
	return s.errorResult(unsupportedRequest(request.Type)), false
}

func okResult(response *streamResponse) streamResult {
	return streamResult{Type: resultTypeOK, Response: response}
}

// executeStatement runs one statement and leaves out its rows when the client does not want them.
func (s *Server) executeStatement(ctx context.Context, input *statement) (*statementResult, error) {
	result, err := s.execute(ctx, input)
	if err != nil {
		return nil, err
	}
	if input.WantRows != nil && !*input.WantRows {
		result.Rows = [][]value{}
	}
	return result, nil
}

// runBatch runs the steps of a batch in order. A failed step does not stop the batch: its error is recorded,
// and the conditions of the later steps can refer to it.
func (s *Server) runBatch(ctx context.Context, input *batch) *batchResult {
	result := &batchResult{
		StepResults: make([]*statementResult, len(input.Steps)),
		StepErrors:  make([]*errorResult, len(input.Steps)),
	}
	for index, step := range input.Steps {
		if step.Condition != nil {
			run, err := step.Condition.evaluate(result)
			if err != nil {
				result.StepErrors[index] = s.errorOf(err)
				continue
			}
			if !run {
				continue
			}
		}
		executed, err := s.executeStatement(ctx, &step.Stmt)
		if err != nil {
			result.StepErrors[index] = s.errorOf(err)
			continue
		}
		result.StepResults[index] = executed
	}
	return result
}

// evaluate tells whether a step runs. A step that has not run yet counts as neither succeeded nor failed.
func (c *batchCondition) evaluate(result *batchResult) (bool, error) {
	switch c.Type {
	case conditionTypeOK, conditionTypeError:
		if c.Step == nil {
			break
		}
		step := int(*c.Step)
		if step >= len(result.StepResults) {
			return false, nil
		}
		if c.Type == conditionTypeOK {
			return result.StepResults[step] != nil, nil
		}
		return result.StepErrors[step] != nil, nil
	case conditionTypeNot:
		if c.Cond == nil {
			break
		}
		inner, err := c.Cond.evaluate(result)
		return !inner, err
	case conditionTypeAnd, conditionTypeOr:
		for _, inner := range c.Conds {
			matched, err := inner.evaluate(result)
			if err != nil {
				return false, err
			}
			// and stops at the first false, or at the first true
			if matched != (c.Type == conditionTypeAnd) {
				return matched, nil
			}
		}
		return c.Type == conditionTypeAnd, nil
	case conditionTypeAutocommit:
		return alwaysAutocommit, nil
	}
	return false, fmt.Errorf(conditionErrorFormat, errors.New(yaagwadb.InvalidArgument), c.Type)
}

// cursorEntries turns the result of a batch into cursor entries. A skipped step has no entry.
func cursorEntries(result *batchResult) []any {
	var entries []any
	for index, executed := range result.StepResults {
		step := uint32(index)
		if failure := result.StepErrors[index]; failure != nil {
			entries = append(entries, &stepErrorEntry{Type: entryTypeStepError, Step: step, Error: failure})
			continue
		}
		if executed == nil {
			continue
		}
		entries = append(entries, &stepBeginEntry{Type: entryTypeStepBegin, Step: step, Columns: executed.Columns})
		for _, row := range executed.Rows {
			entries = append(entries, &rowEntry{Type: entryTypeRow, Row: row})
		}
		entries = append(entries, &stepEndEntry{Type: entryTypeStepEnd, AffectedRowCount: executed.AffectedRowCount,
			LastInsertRowID: executed.LastInsertRowID})
	}
	return entries
}

func (s *Server) errorResult(err error) streamResult {
	return streamResult{Type: resultTypeError, Error: s.errorOf(err)}
}

func (s *Server) errorOf(err error) *errorResult {
	code := errorCode(err)
	if code == nil {
		s.config.Logger.Warn(logRequestFailed, logKeyError, err, logKeyStack, yaagwadb.StackOf(err))
	}
	return &errorResult{Message: err.Error(), Code: code}
}

func errorCode(err error) *string {
	for _, candidate := range errorCodes {
		if yaagwadb.Has(err, candidate.message) {
			code := candidate.code
			return &code
		}
	}
	return nil
}

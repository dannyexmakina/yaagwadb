package hrana

import "encoding/json"

const (
	requestTypeExecute       = "execute"
	requestTypeClose         = "close"
	requestTypeBatch         = "batch"
	requestTypeSequence      = "sequence"
	requestTypeDescribe      = "describe"
	requestTypeStoreSQL      = "store_sql"
	requestTypeCloseSQL      = "close_sql"
	requestTypeGetAutocommit = "get_autocommit"
	resultTypeOK             = "ok"
	resultTypeError          = "error"
	valueTypeNull            = "null"
	valueTypeInteger         = "integer"
	valueTypeFloat           = "float"
	valueTypeText            = "text"
	valueTypeBlob            = "blob"
	conditionTypeOK          = "ok"
	conditionTypeError       = "error"
	conditionTypeNot         = "not"
	conditionTypeAnd         = "and"
	conditionTypeOr          = "or"
	conditionTypeAutocommit  = "is_autocommit"
	entryTypeStepBegin       = "step_begin"
	entryTypeStepEnd         = "step_end"
	entryTypeStepError       = "step_error"
	entryTypeRow             = "row"
	headerContentType        = "Content-Type"
	jsonContentType          = "application/json"
	protobufContentType      = "application/x-protobuf"
	jsonVersionPath          = "/v3"
	protobufVersionPath      = "/v3-protobuf"
	pipelinePathSuffix       = "/pipeline"
	cursorPathSuffix         = "/cursor"
	emptyBaton               = ""
	firstStatementPosition   = 1
	// alwaysAutocommit is the autocommit state of every stream: the SQL front has no transactions.
	alwaysAutocommit = true
)

type pipelineRequest struct {
	Baton    *string         `json:"baton"`
	Requests []streamRequest `json:"requests"`
}

type cursorRequest struct {
	Baton *string `json:"baton"`
	Batch *batch  `json:"batch"`
}

// streamRequest is one request of a pipeline. Stmt is for execute, Batch for batch, and SQL or SQLID
// for sequence, describe, store_sql and close_sql.
type streamRequest struct {
	Type  string     `json:"type"`
	Stmt  *statement `json:"stmt,omitempty"`
	Batch *batch     `json:"batch,omitempty"`
	SQL   *string    `json:"sql,omitempty"`
	SQLID *int64     `json:"sql_id,omitempty"`
}

type statement struct {
	SQL       *string         `json:"sql"`
	SQLID     *int64          `json:"sql_id"`
	Args      []value         `json:"args"`
	NamedArgs []namedArgument `json:"named_args"`
	WantRows  *bool           `json:"want_rows"`
}

type namedArgument struct {
	Name  string `json:"name"`
	Value value  `json:"value"`
}

// value is a Hrana value. Value holds the integer (as a decimal string), the float, or the text; Base64 holds a blob.
type value struct {
	Type   string          `json:"type"`
	Value  json.RawMessage `json:"value,omitempty"`
	Base64 *string         `json:"base64,omitempty"`
}

type batch struct {
	Steps []batchStep `json:"steps"`
}

type batchStep struct {
	Condition *batchCondition `json:"condition"`
	Stmt      statement       `json:"stmt"`
}

// batchCondition decides whether a step runs. Step is for ok and error, Cond for not, Conds for and and or.
type batchCondition struct {
	Type  string           `json:"type"`
	Step  *uint32          `json:"step,omitempty"`
	Cond  *batchCondition  `json:"cond,omitempty"`
	Conds []batchCondition `json:"conds,omitempty"`
}

type pipelineResponse struct {
	Baton   *string        `json:"baton"`
	BaseURL *string        `json:"base_url"`
	Results []streamResult `json:"results"`
}

type cursorResponse struct {
	Baton   *string `json:"baton"`
	BaseURL *string `json:"base_url"`
}

type streamResult struct {
	Type     string          `json:"type"`
	Response *streamResponse `json:"response,omitempty"`
	Error    *errorResult    `json:"error,omitempty"`
}

// streamResponse is the response to one request. Result is a *statementResult for execute,
// a *batchResult for batch and a *describeResult for describe.
type streamResponse struct {
	Type         string `json:"type"`
	Result       any    `json:"result,omitempty"`
	IsAutocommit *bool  `json:"is_autocommit,omitempty"`
}

type errorResult struct {
	Message string  `json:"message"`
	Code    *string `json:"code"`
}

// statementResult is the result of one statement. The SQL front does not count the rows read and written
// or the time a statement takes, so those are always zero.
type statementResult struct {
	Columns                   []column  `json:"cols"`
	Rows                      [][]value `json:"rows"`
	AffectedRowCount          int64     `json:"affected_row_count"`
	LastInsertRowID           *string   `json:"last_insert_rowid"`
	RowsRead                  int64     `json:"rows_read"`
	RowsWritten               int64     `json:"rows_written"`
	QueryDurationMilliseconds float64   `json:"query_duration_ms"`
}

type column struct {
	Name     string  `json:"name"`
	Decltype *string `json:"decltype"`
}

// batchResult holds, for each step, the result or the error of its statement. Both are nil for a skipped step.
type batchResult struct {
	StepResults []*statementResult `json:"step_results"`
	StepErrors  []*errorResult     `json:"step_errors"`
}

type describeResult struct {
	Params     []describeParameter `json:"params"`
	Columns    []column            `json:"cols"`
	IsExplain  bool                `json:"is_explain"`
	IsReadonly bool                `json:"is_readonly"`
}

// describeParameter is a parameter of a statement. Name is nil for a parameter written as a bare ?.
type describeParameter struct {
	Name *string `json:"name"`
}

type stepBeginEntry struct {
	Type    string   `json:"type"`
	Step    uint32   `json:"step"`
	Columns []column `json:"cols"`
}

type stepEndEntry struct {
	Type             string  `json:"type"`
	AffectedRowCount int64   `json:"affected_row_count"`
	LastInsertRowID  *string `json:"last_insert_rowid"`
}

type stepErrorEntry struct {
	Type  string       `json:"type"`
	Step  uint32       `json:"step"`
	Error *errorResult `json:"error"`
}

type rowEntry struct {
	Type string  `json:"type"`
	Row  []value `json:"row"`
}

package hrana

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"yaagwadb"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protowire"
)

// Field numbers of the Protobuf schema of Hrana 3 (HRANA_3_SPEC.md, appendix), by message.
const (
	// PipelineReqBody, PipelineRespBody, CursorReqBody, CursorRespBody
	fieldBaton            = 1
	fieldBaseURL          = 2
	fieldPipelineRequests = 2
	fieldPipelineResults  = 3
	fieldCursorBatch      = 2
	// StreamResult
	fieldResultOK    = 1
	fieldResultError = 2
	// ExecuteStreamReq, BatchStreamReq, SequenceStreamReq, DescribeStreamReq, StoreSqlStreamReq, CloseSqlStreamReq
	fieldExecuteStatement = 1
	fieldBatchRequest     = 1
	fieldSQLText          = 1
	fieldSQLTextID        = 2
	fieldStoreSQLID       = 1
	fieldStoreSQLText     = 2
	fieldCloseSQLID       = 1
	// ExecuteStreamResp, BatchStreamResp, DescribeStreamResp, GetAutocommitStreamResp
	fieldResponseResult       = 1
	fieldResponseIsAutocommit = 1
	// Error
	fieldErrorMessage = 1
	fieldErrorCode    = 2
	// Stmt
	fieldStatementSQL       = 1
	fieldStatementSQLID     = 2
	fieldStatementArgs      = 3
	fieldStatementNamedArgs = 4
	fieldStatementWantRows  = 5
	// NamedArg
	fieldNamedArgumentName  = 1
	fieldNamedArgumentValue = 2
	// Value
	fieldValueNull    = 1
	fieldValueInteger = 2
	fieldValueFloat   = 3
	fieldValueText    = 4
	fieldValueBlob    = 5
	// StmtResult
	fieldStatementResultColumns          = 1
	fieldStatementResultRows             = 2
	fieldStatementResultAffectedRowCount = 3
	fieldStatementResultLastInsertRowID  = 4
	// Col, DescribeCol
	fieldColumnName     = 1
	fieldColumnDecltype = 2
	// Row
	fieldRowValues = 1
	// Batch, BatchStep
	fieldBatchSteps         = 1
	fieldBatchStepCondition = 1
	fieldBatchStepStatement = 2
	// BatchCond, BatchCond.CondList
	fieldConditionStepOK     = 1
	fieldConditionStepError  = 2
	fieldConditionNot        = 3
	fieldConditionAnd        = 4
	fieldConditionOr         = 5
	fieldConditionAutocommit = 6
	fieldConditionListConds  = 1
	// BatchResult and its map entries
	fieldBatchResultStepResults = 1
	fieldBatchResultStepErrors  = 2
	fieldMapKey                 = 1
	fieldMapValue               = 2
	// DescribeResult, DescribeParam
	fieldDescribeParams     = 1
	fieldDescribeColumns    = 2
	fieldDescribeIsExplain  = 3
	fieldDescribeIsReadonly = 4
	fieldParameterName      = 1
	// CursorEntry, StepBeginEntry, StepEndEntry, StepErrorEntry
	fieldEntryStepBegin          = 1
	fieldEntryStepEnd            = 2
	fieldEntryStepError          = 3
	fieldEntryRow                = 4
	fieldStepBeginStep           = 1
	fieldStepBeginColumns        = 2
	fieldStepEndAffectedRowCount = 1
	fieldStepEndLastInsertRowID  = 2
	fieldStepErrorStep           = 1
	fieldStepErrorError          = 2
)

// streamRequestTypes gives the request type of each field of StreamRequest. StreamResponse uses the same fields.
var streamRequestTypes = map[protowire.Number]string{
	1: requestTypeClose,
	2: requestTypeExecute,
	3: requestTypeBatch,
	4: requestTypeSequence,
	5: requestTypeDescribe,
	6: requestTypeStoreSQL,
	7: requestTypeCloseSQL,
	8: requestTypeGetAutocommit,
}

const (
	protobufErrorFormat    = "%w: protobuf: %v"
	protobufWireTypeFormat = "field %d has wire type %d"
	protobufEncodeFormat   = "hrana: cannot encode %v in protobuf"
)

func protobufError(err error) error {
	return fmt.Errorf(protobufErrorFormat, errors.New(yaagwadb.InvalidArgument), err)
}

// eachField calls visit with the number, the wire type and the encoded value of each field of a message.
func eachField(message []byte, visit func(protowire.Number, protowire.Type, []byte) error) error {
	for len(message) > 0 {
		number, kind, length := protowire.ConsumeTag(message)
		if length < 0 {
			return protobufError(protowire.ParseError(length))
		}
		message = message[length:]
		size := protowire.ConsumeFieldValue(number, kind, message)
		if size < 0 {
			return protobufError(protowire.ParseError(size))
		}
		if err := visit(number, kind, message[:size]); err != nil {
			return err
		}
		message = message[size:]
	}
	return nil
}

func wireTypeError(number protowire.Number, kind protowire.Type) error {
	return protobufError(fmt.Errorf(protobufWireTypeFormat, number, kind))
}

func bytesField(number protowire.Number, kind protowire.Type, field []byte) ([]byte, error) {
	if kind != protowire.BytesType {
		return nil, wireTypeError(number, kind)
	}
	content, length := protowire.ConsumeBytes(field)
	if length < 0 {
		return nil, protobufError(protowire.ParseError(length))
	}
	return content, nil
}

func varintField(number protowire.Number, kind protowire.Type, field []byte) (uint64, error) {
	if kind != protowire.VarintType {
		return 0, wireTypeError(number, kind)
	}
	content, length := protowire.ConsumeVarint(field)
	if length < 0 {
		return 0, protobufError(protowire.ParseError(length))
	}
	return content, nil
}

func stringField(number protowire.Number, kind protowire.Type, field []byte) (*string, error) {
	content, err := bytesField(number, kind, field)
	if err != nil {
		return nil, err
	}
	text := string(content)
	return &text, nil
}

// integerField reads an int32 or int64 field, which is a varint of the value extended to 64 bits.
func integerField(number protowire.Number, kind protowire.Type, field []byte) (*int64, error) {
	content, err := varintField(number, kind, field)
	if err != nil {
		return nil, err
	}
	integer := int64(content)
	return &integer, nil
}

func readProtobufPipeline(body []byte) (*pipelineRequest, error) {
	request := &pipelineRequest{}
	err := eachField(body, func(number protowire.Number, kind protowire.Type, field []byte) error {
		switch number {
		case fieldBaton:
			baton, err := stringField(number, kind, field)
			request.Baton = baton
			return err
		case fieldPipelineRequests:
			message, err := bytesField(number, kind, field)
			if err != nil {
				return err
			}
			decoded, err := readStreamRequest(message)
			if err != nil {
				return err
			}
			request.Requests = append(request.Requests, *decoded)
		}
		return nil
	})
	return request, err
}

func readProtobufCursor(body []byte) (*cursorRequest, error) {
	request := &cursorRequest{}
	err := eachField(body, func(number protowire.Number, kind protowire.Type, field []byte) error {
		switch number {
		case fieldBaton:
			baton, err := stringField(number, kind, field)
			request.Baton = baton
			return err
		case fieldCursorBatch:
			message, err := bytesField(number, kind, field)
			if err != nil {
				return err
			}
			request.Batch, err = readBatch(message)
			return err
		}
		return nil
	})
	return request, err
}

// readStreamRequest reads a StreamRequest, whose one field is the request of that field's type.
func readStreamRequest(message []byte) (*streamRequest, error) {
	request := &streamRequest{}
	err := eachField(message, func(number protowire.Number, kind protowire.Type, field []byte) error {
		requestType, known := streamRequestTypes[number]
		if !known {
			return nil
		}
		body, err := bytesField(number, kind, field)
		if err != nil {
			return err
		}
		request.Type = requestType
		return readStreamRequestBody(request, body)
	})
	return request, err
}

func readStreamRequestBody(request *streamRequest, body []byte) error {
	return eachField(body, func(number protowire.Number, kind protowire.Type, field []byte) error {
		var err error
		switch request.Type {
		case requestTypeExecute:
			if number == fieldExecuteStatement {
				var message []byte
				if message, err = bytesField(number, kind, field); err == nil {
					request.Stmt, err = readStatement(message)
				}
			}
		case requestTypeBatch:
			if number == fieldBatchRequest {
				var message []byte
				if message, err = bytesField(number, kind, field); err == nil {
					request.Batch, err = readBatch(message)
				}
			}
		case requestTypeSequence, requestTypeDescribe:
			switch number {
			case fieldSQLText:
				request.SQL, err = stringField(number, kind, field)
			case fieldSQLTextID:
				request.SQLID, err = integerField(number, kind, field)
			}
		case requestTypeStoreSQL:
			switch number {
			case fieldStoreSQLID:
				request.SQLID, err = integerField(number, kind, field)
			case fieldStoreSQLText:
				request.SQL, err = stringField(number, kind, field)
			}
		case requestTypeCloseSQL:
			if number == fieldCloseSQLID {
				request.SQLID, err = integerField(number, kind, field)
			}
		}
		return err
	})
}

func readStatement(message []byte) (*statement, error) {
	input := &statement{}
	err := eachField(message, func(number protowire.Number, kind protowire.Type, field []byte) error {
		var err error
		switch number {
		case fieldStatementSQL:
			input.SQL, err = stringField(number, kind, field)
		case fieldStatementSQLID:
			input.SQLID, err = integerField(number, kind, field)
		case fieldStatementArgs:
			var argument []byte
			if argument, err = bytesField(number, kind, field); err == nil {
				var decoded value
				if decoded, err = readValue(argument); err == nil {
					input.Args = append(input.Args, decoded)
				}
			}
		case fieldStatementNamedArgs:
			var argument []byte
			if argument, err = bytesField(number, kind, field); err == nil {
				var decoded namedArgument
				if decoded, err = readNamedArgument(argument); err == nil {
					input.NamedArgs = append(input.NamedArgs, decoded)
				}
			}
		case fieldStatementWantRows:
			var flag uint64
			if flag, err = varintField(number, kind, field); err == nil {
				wanted := protowire.DecodeBool(flag)
				input.WantRows = &wanted
			}
		}
		return err
	})
	return input, err
}

func readNamedArgument(message []byte) (namedArgument, error) {
	var argument namedArgument
	err := eachField(message, func(number protowire.Number, kind protowire.Type, field []byte) error {
		switch number {
		case fieldNamedArgumentName:
			name, err := stringField(number, kind, field)
			if err != nil {
				return err
			}
			argument.Name = *name
		case fieldNamedArgumentValue:
			content, err := bytesField(number, kind, field)
			if err != nil {
				return err
			}
			argument.Value, err = readValue(content)
			return err
		}
		return nil
	})
	if argument.Value.Type == "" {
		argument.Value = value{Type: valueTypeNull}
	}
	return argument, err
}

// readValue reads a Value into the form of its JSON encoding. A Value with none of its fields set is null.
func readValue(message []byte) (value, error) {
	decoded := value{Type: valueTypeNull}
	err := eachField(message, func(number protowire.Number, kind protowire.Type, field []byte) error {
		switch number {
		case fieldValueNull:
			decoded = value{Type: valueTypeNull}
		case fieldValueInteger:
			content, err := varintField(number, kind, field)
			if err != nil {
				return err
			}
			decoded = jsonValue(valueTypeInteger, strconv.FormatInt(protowire.DecodeZigZag(content), decimalBase))
		case fieldValueFloat:
			if kind != protowire.Fixed64Type {
				return wireTypeError(number, kind)
			}
			content, length := protowire.ConsumeFixed64(field)
			if length < 0 {
				return protobufError(protowire.ParseError(length))
			}
			decoded = jsonValue(valueTypeFloat, math.Float64frombits(content))
		case fieldValueText:
			content, err := bytesField(number, kind, field)
			if err != nil {
				return err
			}
			decoded = textValue(string(content))
		case fieldValueBlob:
			content, err := bytesField(number, kind, field)
			if err != nil {
				return err
			}
			encoded := base64.StdEncoding.EncodeToString(content)
			decoded = value{Type: valueTypeBlob, Base64: &encoded}
		}
		return nil
	})
	return decoded, err
}

func jsonValue(valueType string, content any) value {
	encoded, _ := json.Marshal(content)
	return value{Type: valueType, Value: encoded}
}

func readBatch(message []byte) (*batch, error) {
	decoded := &batch{}
	err := eachField(message, func(number protowire.Number, kind protowire.Type, field []byte) error {
		if number != fieldBatchSteps {
			return nil
		}
		content, err := bytesField(number, kind, field)
		if err != nil {
			return err
		}
		step, err := readBatchStep(content)
		if err != nil {
			return err
		}
		decoded.Steps = append(decoded.Steps, step)
		return nil
	})
	return decoded, err
}

func readBatchStep(message []byte) (batchStep, error) {
	var step batchStep
	err := eachField(message, func(number protowire.Number, kind protowire.Type, field []byte) error {
		var err error
		switch number {
		case fieldBatchStepCondition:
			var content []byte
			if content, err = bytesField(number, kind, field); err == nil {
				step.Condition, err = readCondition(content)
			}
		case fieldBatchStepStatement:
			var content []byte
			if content, err = bytesField(number, kind, field); err == nil {
				var decoded *statement
				if decoded, err = readStatement(content); err == nil {
					step.Stmt = *decoded
				}
			}
		}
		return err
	})
	return step, err
}

// readCondition reads a BatchCond. One without a known field has no type, which evaluating it refuses.
func readCondition(message []byte) (*batchCondition, error) {
	condition := &batchCondition{}
	err := eachField(message, func(number protowire.Number, kind protowire.Type, field []byte) error {
		switch number {
		case fieldConditionStepOK, fieldConditionStepError:
			content, err := varintField(number, kind, field)
			if err != nil {
				return err
			}
			step := uint32(content)
			conditionType := conditionTypeOK
			if number == fieldConditionStepError {
				conditionType = conditionTypeError
			}
			*condition = batchCondition{Type: conditionType, Step: &step}
		case fieldConditionNot:
			content, err := bytesField(number, kind, field)
			if err != nil {
				return err
			}
			inner, err := readCondition(content)
			if err != nil {
				return err
			}
			*condition = batchCondition{Type: conditionTypeNot, Cond: inner}
		case fieldConditionAnd, fieldConditionOr:
			content, err := bytesField(number, kind, field)
			if err != nil {
				return err
			}
			conditions, err := readConditionList(content)
			if err != nil {
				return err
			}
			conditionType := conditionTypeAnd
			if number == fieldConditionOr {
				conditionType = conditionTypeOr
			}
			*condition = batchCondition{Type: conditionType, Conds: conditions}
		case fieldConditionAutocommit:
			*condition = batchCondition{Type: conditionTypeAutocommit}
		}
		return nil
	})
	return condition, err
}

func readConditionList(message []byte) ([]batchCondition, error) {
	conditions := []batchCondition{}
	err := eachField(message, func(number protowire.Number, kind protowire.Type, field []byte) error {
		if number != fieldConditionListConds {
			return nil
		}
		content, err := bytesField(number, kind, field)
		if err != nil {
			return err
		}
		inner, err := readCondition(content)
		if err != nil {
			return err
		}
		conditions = append(conditions, *inner)
		return nil
	})
	return conditions, err
}

func appendMessageField(message []byte, number protowire.Number, content []byte) []byte {
	message = protowire.AppendTag(message, number, protowire.BytesType)
	return protowire.AppendBytes(message, content)
}

func appendStringField(message []byte, number protowire.Number, text string) []byte {
	message = protowire.AppendTag(message, number, protowire.BytesType)
	return protowire.AppendString(message, text)
}

func appendVarintField(message []byte, number protowire.Number, content uint64) []byte {
	message = protowire.AppendTag(message, number, protowire.VarintType)
	return protowire.AppendVarint(message, content)
}

func appendBoolField(message []byte, number protowire.Number, flag bool) []byte {
	return appendVarintField(message, number, protowire.EncodeBool(flag))
}

// appendLastInsertRowID appends the optional sint64 last_insert_rowid, which the JSON encoding holds as a decimal string.
func appendLastInsertRowID(message []byte, number protowire.Number, rowID *string) []byte {
	if rowID == nil {
		return message
	}
	integer, err := strconv.ParseInt(*rowID, decimalBase, integerBitSize)
	if err != nil {
		return message
	}
	return appendVarintField(message, number, protowire.EncodeZigZag(integer))
}

func writeProtobufPipeline(response *pipelineResponse) ([]byte, error) {
	var message []byte
	if response.Baton != nil {
		message = appendStringField(message, fieldBaton, *response.Baton)
	}
	if response.BaseURL != nil {
		message = appendStringField(message, fieldBaseURL, *response.BaseURL)
	}
	for _, result := range response.Results {
		encoded, err := encodeStreamResult(result)
		if err != nil {
			return nil, err
		}
		message = appendMessageField(message, fieldPipelineResults, encoded)
	}
	return message, nil
}

// writeProtobufCursor writes a CursorRespBody prefixed with its length, as the cursor response separates its structures.
func writeProtobufCursor(response *cursorResponse) ([]byte, error) {
	var message []byte
	if response.Baton != nil {
		message = appendStringField(message, fieldBaton, *response.Baton)
	}
	if response.BaseURL != nil {
		message = appendStringField(message, fieldBaseURL, *response.BaseURL)
	}
	return protowire.AppendBytes(nil, message), nil
}

// writeProtobufEntry writes a CursorEntry prefixed with its length.
func writeProtobufEntry(entry any) ([]byte, error) {
	var message []byte
	switch typed := entry.(type) {
	case *stepBeginEntry:
		body := appendVarintField(nil, fieldStepBeginStep, uint64(typed.Step))
		for _, item := range typed.Columns {
			body = appendMessageField(body, fieldStepBeginColumns, encodeColumn(item))
		}
		message = appendMessageField(nil, fieldEntryStepBegin, body)
	case *stepEndEntry:
		body := appendVarintField(nil, fieldStepEndAffectedRowCount, uint64(typed.AffectedRowCount))
		body = appendLastInsertRowID(body, fieldStepEndLastInsertRowID, typed.LastInsertRowID)
		message = appendMessageField(nil, fieldEntryStepEnd, body)
	case *stepErrorEntry:
		body := appendVarintField(nil, fieldStepErrorStep, uint64(typed.Step))
		body = appendMessageField(body, fieldStepErrorError, encodeError(typed.Error))
		message = appendMessageField(nil, fieldEntryStepError, body)
	case *rowEntry:
		message = appendMessageField(nil, fieldEntryRow, encodeRow(typed.Row))
	default:
		return nil, fmt.Errorf(protobufEncodeFormat, entry)
	}
	return protowire.AppendBytes(nil, message), nil
}

func encodeStreamResult(result streamResult) ([]byte, error) {
	if result.Type == resultTypeError {
		return appendMessageField(nil, fieldResultError, encodeError(result.Error)), nil
	}
	response, err := encodeStreamResponse(result.Response)
	if err != nil {
		return nil, err
	}
	return appendMessageField(nil, fieldResultOK, response), nil
}

// encodeStreamResponse writes a StreamResponse, whose one field is the response of that field's type.
func encodeStreamResponse(response *streamResponse) ([]byte, error) {
	var body []byte
	switch typed := response.Result.(type) {
	case nil:
	case *statementResult:
		body = appendMessageField(body, fieldResponseResult, encodeStatementResult(typed))
	case *batchResult:
		body = appendMessageField(body, fieldResponseResult, encodeBatchResult(typed))
	case *describeResult:
		body = appendMessageField(body, fieldResponseResult, encodeDescribeResult(typed))
	default:
		return nil, fmt.Errorf(protobufEncodeFormat, response.Result)
	}
	if response.IsAutocommit != nil {
		body = appendBoolField(body, fieldResponseIsAutocommit, *response.IsAutocommit)
	}
	for number, requestType := range streamRequestTypes {
		if requestType == response.Type {
			return appendMessageField(nil, number, body), nil
		}
	}
	return nil, fmt.Errorf(protobufEncodeFormat, response.Type)
}

func encodeError(failure *errorResult) []byte {
	if failure == nil {
		return nil
	}
	message := appendStringField(nil, fieldErrorMessage, failure.Message)
	if failure.Code != nil {
		message = appendStringField(message, fieldErrorCode, *failure.Code)
	}
	return message
}

func encodeStatementResult(result *statementResult) []byte {
	var message []byte
	for _, item := range result.Columns {
		message = appendMessageField(message, fieldStatementResultColumns, encodeColumn(item))
	}
	for _, row := range result.Rows {
		message = appendMessageField(message, fieldStatementResultRows, encodeRow(row))
	}
	message = appendVarintField(message, fieldStatementResultAffectedRowCount, uint64(result.AffectedRowCount))
	return appendLastInsertRowID(message, fieldStatementResultLastInsertRowID, result.LastInsertRowID)
}

func encodeColumn(item column) []byte {
	message := appendStringField(nil, fieldColumnName, item.Name)
	if item.Decltype != nil {
		message = appendStringField(message, fieldColumnDecltype, *item.Decltype)
	}
	return message
}

func encodeRow(row []value) []byte {
	var message []byte
	for _, item := range row {
		message = appendMessageField(message, fieldRowValues, encodeValue(item))
	}
	return message
}

// encodeValue writes a Value from the form of its JSON encoding. A value that cannot be read is written as null.
func encodeValue(input value) []byte {
	switch input.Type {
	case valueTypeInteger:
		var text string
		if json.Unmarshal(input.Value, &text) == nil {
			if integer, err := strconv.ParseInt(text, decimalBase, integerBitSize); err == nil {
				return appendVarintField(nil, fieldValueInteger, protowire.EncodeZigZag(integer))
			}
		}
	case valueTypeFloat:
		var real float64
		if json.Unmarshal(input.Value, &real) == nil {
			message := protowire.AppendTag(nil, fieldValueFloat, protowire.Fixed64Type)
			return protowire.AppendFixed64(message, math.Float64bits(real))
		}
	case valueTypeText:
		var text string
		if json.Unmarshal(input.Value, &text) == nil {
			return appendStringField(nil, fieldValueText, text)
		}
	case valueTypeBlob:
		if input.Base64 != nil {
			if content, err := base64.StdEncoding.DecodeString(*input.Base64); err == nil {
				message := protowire.AppendTag(nil, fieldValueBlob, protowire.BytesType)
				return protowire.AppendBytes(message, content)
			}
		}
	}
	return appendMessageField(nil, fieldValueNull, nil)
}

// encodeBatchResult writes the results and the errors of the steps as maps from the step to the result or the error.
func encodeBatchResult(result *batchResult) []byte {
	var message []byte
	for step, executed := range result.StepResults {
		if executed == nil {
			continue
		}
		entry := appendVarintField(nil, fieldMapKey, uint64(step))
		entry = appendMessageField(entry, fieldMapValue, encodeStatementResult(executed))
		message = appendMessageField(message, fieldBatchResultStepResults, entry)
	}
	for step, failure := range result.StepErrors {
		if failure == nil {
			continue
		}
		entry := appendVarintField(nil, fieldMapKey, uint64(step))
		entry = appendMessageField(entry, fieldMapValue, encodeError(failure))
		message = appendMessageField(message, fieldBatchResultStepErrors, entry)
	}
	return message
}

func encodeDescribeResult(result *describeResult) []byte {
	var message []byte
	for _, parameter := range result.Params {
		var content []byte
		if parameter.Name != nil {
			content = appendStringField(nil, fieldParameterName, *parameter.Name)
		}
		message = appendMessageField(message, fieldDescribeParams, content)
	}
	for _, item := range result.Columns {
		message = appendMessageField(message, fieldDescribeColumns, encodeColumn(item))
	}
	message = appendBoolField(message, fieldDescribeIsExplain, result.IsExplain)
	return appendBoolField(message, fieldDescribeIsReadonly, result.IsReadonly)
}

package hrana

import (
	"encoding/base64"
	"math"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

const (
	testExecuteField         = 2
	testBatchField           = 3
	testGetAutocommitField   = 8
	testCloseField           = 1
	testBaton                = "baton"
	testInteger64            = -9007199254740993
	testInteger64Text        = `"-9007199254740993"`
	testFloat                = 0.25
	testFloatText            = "0.25"
	testSecondStep           = 1
	testUnknownConditionType = "unknown"
	testProtobufFormat       = "%s: %+v"
	testRowID                = "-12"
	testRowIDValue           = -12
	testWrongWireType        = "wrong wire type"
)

func TestReadProtobufPipeline(t *testing.T) {
	arguments := appendVarintField(nil, fieldValueInteger, protowire.EncodeZigZag(testInteger64))
	floatValue := protowire.AppendFixed64(protowire.AppendTag(nil, fieldValueFloat, protowire.Fixed64Type), math.Float64bits(testFloat))
	blobValue := appendMessageField(nil, fieldValueBlob, []byte{0, 255})
	statementMessage := appendStringField(nil, fieldStatementSQL, testSelectPaths)
	statementMessage = appendMessageField(statementMessage, fieldStatementArgs, arguments)
	statementMessage = appendMessageField(statementMessage, fieldStatementArgs, floatValue)
	statementMessage = appendMessageField(statementMessage, fieldStatementArgs, blobValue)
	statementMessage = appendMessageField(statementMessage, fieldStatementArgs, nil)
	statementMessage = appendBoolField(statementMessage, fieldStatementWantRows, false)
	condition := appendMessageField(nil, fieldConditionNot, appendVarintField(nil, fieldConditionStepError, testSecondStep))
	step := appendMessageField(nil, fieldBatchStepCondition, condition)
	step = appendMessageField(step, fieldBatchStepStatement, statementMessage)

	body := appendStringField(nil, fieldBaton, testBaton)
	body = appendMessageField(body, fieldPipelineRequests,
		appendMessageField(nil, testExecuteField, appendMessageField(nil, fieldExecuteStatement, statementMessage)))
	body = appendMessageField(body, fieldPipelineRequests,
		appendMessageField(nil, testBatchField, appendMessageField(nil, fieldBatchRequest, appendMessageField(nil, fieldBatchSteps, step))))
	body = appendMessageField(body, fieldPipelineRequests, appendMessageField(nil, testGetAutocommitField, nil))

	request, err := readProtobufPipeline(body)
	if err != nil {
		t.Fatal(err)
	}
	if *request.Baton != testBaton || len(request.Requests) != 3 {
		t.Fatalf(testProtobufFormat, testBaton, request)
	}
	execute := request.Requests[0]
	args := execute.Stmt.Args
	if execute.Type != requestTypeExecute || *execute.Stmt.SQL != testSelectPaths || *execute.Stmt.WantRows ||
		args[0].Type != valueTypeInteger || string(args[0].Value) != testInteger64Text ||
		args[1].Type != valueTypeFloat || string(args[1].Value) != testFloatText ||
		args[2].Type != valueTypeBlob || *args[2].Base64 != base64.StdEncoding.EncodeToString([]byte{0, 255}) ||
		args[3].Type != valueTypeNull {
		t.Fatalf(testProtobufFormat, requestTypeExecute, execute)
	}
	steps := request.Requests[1].Batch.Steps
	if request.Requests[1].Type != requestTypeBatch || len(steps) != 1 || steps[0].Condition.Type != conditionTypeNot ||
		steps[0].Condition.Cond.Type != conditionTypeError || *steps[0].Condition.Cond.Step != testSecondStep ||
		*steps[0].Stmt.SQL != testSelectPaths {
		t.Fatalf(testProtobufFormat, requestTypeBatch, steps)
	}
	if request.Requests[2].Type != requestTypeGetAutocommit {
		t.Fatalf(testProtobufFormat, requestTypeGetAutocommit, request.Requests[2])
	}
}

func TestWriteProtobufPipeline(t *testing.T) {
	baton := testBaton
	rowID := testRowID
	autocommit := true
	encodedResponse, err := writeProtobufPipeline(&pipelineResponse{Baton: &baton, Results: []streamResult{
		okResult(&streamResponse{Type: requestTypeExecute, Result: &statementResult{
			Columns: []column{{Name: columnPath}}, Rows: [][]value{{textValue(testPath)}}, LastInsertRowID: &rowID}}),
		okResult(&streamResponse{Type: requestTypeGetAutocommit, IsAutocommit: &autocommit}),
		okResult(&streamResponse{Type: requestTypeClose}),
		{Type: resultTypeError, Error: &errorResult{Message: testPath}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	fields := fieldsOf(t, encodedResponse)
	if string(fields[fieldBaton][0]) != testBaton || len(fields[fieldPipelineResults]) != 4 {
		t.Fatalf(testProtobufFormat, testBaton, fields)
	}
	results := fields[fieldPipelineResults]
	execute := fieldsOf(t, fieldsOf(t, fieldsOf(t, results[0])[fieldResultOK][0])[testExecuteField][0])[fieldResponseResult][0]
	row := fieldsOf(t, fieldsOf(t, execute)[fieldStatementResultRows][0])[fieldRowValues][0]
	if string(fieldsOf(t, row)[fieldValueText][0]) != testPath {
		t.Fatalf(testProtobufFormat, requestTypeExecute, execute)
	}
	var rowIDValue uint64
	_ = eachField(execute, func(number protowire.Number, kind protowire.Type, field []byte) error {
		if number == fieldStatementResultLastInsertRowID {
			rowIDValue, _ = varintField(number, kind, field)
		}
		return nil
	})
	if protowire.DecodeZigZag(rowIDValue) != testRowIDValue {
		t.Fatalf(testProtobufFormat, testRowID, rowIDValue)
	}
	if len(fieldsOf(t, results[1])[fieldResultOK]) != 1 || len(fieldsOf(t, fieldsOf(t, results[2])[fieldResultOK][0])[testCloseField]) != 1 ||
		string(fieldsOf(t, fieldsOf(t, results[3])[fieldResultError][0])[fieldErrorMessage][0]) != testPath {
		t.Fatalf(testProtobufFormat, resultTypeError, results)
	}
}

func TestReadProtobufRejectsWrongWireType(t *testing.T) {
	body := appendVarintField(nil, fieldPipelineRequests, 1)
	if _, err := readProtobufPipeline(body); errorCode(err) == nil || *errorCode(err) != errorCodeInvalidArgument {
		t.Fatalf(testProtobufFormat, testWrongWireType, err)
	}
}

func TestConditionEvaluate(t *testing.T) {
	first, second := uint32(0), uint32(testSecondStep)
	result := &batchResult{StepResults: []*statementResult{{}, nil}, StepErrors: []*errorResult{nil, {}}}
	for _, check := range []struct {
		condition batchCondition
		want      bool
	}{
		{batchCondition{Type: conditionTypeOK, Step: &first}, true},
		{batchCondition{Type: conditionTypeOK, Step: &second}, false},
		{batchCondition{Type: conditionTypeError, Step: &second}, true},
		{batchCondition{Type: conditionTypeNot, Cond: &batchCondition{Type: conditionTypeError, Step: &first}}, true},
		{batchCondition{Type: conditionTypeAnd}, true},
		{batchCondition{Type: conditionTypeOr}, false},
		{batchCondition{Type: conditionTypeOr, Conds: []batchCondition{{Type: conditionTypeOK, Step: &second}, {Type: conditionTypeAutocommit}}}, true},
	} {
		if got, err := check.condition.evaluate(result); err != nil || got != check.want {
			t.Errorf(testProtobufFormat, check.condition.Type, got)
		}
	}
	unknown := batchCondition{Type: testUnknownConditionType}
	if _, err := unknown.evaluate(result); errorCode(err) == nil {
		t.Errorf(testProtobufFormat, testUnknownConditionType, err)
	}
}

func TestDescribe(t *testing.T) {
	query := `SELECT path, size AS s, length( path ), * FROM objects WHERE path = ? AND size > :min AND mtime < ?4 AND xattr = :min`
	result, err := describe(&query)
	if err != nil {
		t.Fatal(err)
	}
	wantParameters := []string{"", ":min", "", "?4"}
	wantColumns := []string{columnPath, "s", "length( path )", columnPath, columnSize, columnMtime, columnDeleted, columnXattr}
	wantDeclaredTypes := []string{"", declaredTypeInteger, "", "", declaredTypeInteger, declaredTypeInteger, "", ""}
	if !result.IsReadonly || len(result.Params) != len(wantParameters) || len(result.Columns) != len(wantColumns) {
		t.Fatalf(testProtobufFormat, query, result)
	}
	for index, want := range wantParameters {
		got := ""
		if result.Params[index].Name != nil {
			got = *result.Params[index].Name
		}
		if got != want {
			t.Fatalf(testProtobufFormat, query, result.Params)
		}
	}
	for index, want := range wantColumns {
		declared := ""
		if result.Columns[index].Decltype != nil {
			declared = *result.Columns[index].Decltype
		}
		if result.Columns[index].Name != want || declared != wantDeclaredTypes[index] {
			t.Fatalf(testProtobufFormat, query, result.Columns)
		}
	}
	insert := testInsert
	if result, err := describe(&insert); err != nil || result.IsReadonly || len(result.Params) != 2 || len(result.Columns) != 0 {
		t.Fatalf(testProtobufFormat, insert, result)
	}
}

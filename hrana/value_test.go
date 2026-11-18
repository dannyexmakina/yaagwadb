package hrana

import (
	"encoding/json"
	"testing"

	"yaagwadb"
)

const (
	testIntegerText       = "42"
	testInteger           = 42
	testReal              = 1.5
	testNotAnInteger      = "x"
	testUnknownValueType  = "date"
	testValueResultFormat = "%+v: got %+v %v"
)

func rawJSON(t *testing.T, input any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestToQueryValue(t *testing.T) {
	integer, err := toQueryValue(value{Type: valueTypeInteger, Value: rawJSON(t, testIntegerText)})
	if err != nil || integer.Integer == nil || *integer.Integer != testInteger {
		t.Fatalf(testValueResultFormat, valueTypeInteger, integer, err)
	}
	real, err := toQueryValue(value{Type: valueTypeFloat, Value: rawJSON(t, testReal)})
	if err != nil || real.Real == nil || *real.Real != testReal {
		t.Fatalf(testValueResultFormat, valueTypeFloat, real, err)
	}
	null, err := toQueryValue(value{Type: valueTypeNull})
	if err != nil || null != (yaagwadb.QueryValue{}) {
		t.Fatalf(testValueResultFormat, valueTypeNull, null, err)
	}
	for _, refused := range []value{
		{Type: valueTypeBlob},
		{Type: testUnknownValueType},
		{Type: valueTypeInteger, Value: rawJSON(t, testNotAnInteger)},
	} {
		if _, err := toQueryValue(refused); errorCode(err) == nil {
			t.Errorf(testValueResultFormat, refused, nil, err)
		}
	}
}

func TestFromQueryValue(t *testing.T) {
	integer := int64(testInteger)
	converted := fromQueryValue(yaagwadb.QueryValue{Integer: &integer})
	if converted.Type != valueTypeInteger || textOf(converted) != testIntegerText {
		t.Fatalf(testValueResultFormat, integer, converted, nil)
	}
	if converted := fromQueryValue(yaagwadb.QueryValue{}); converted.Type != valueTypeNull {
		t.Fatalf(testValueResultFormat, nil, converted, nil)
	}
}

package hrana

import (
	"encoding/json"
	"fmt"
	"strconv"

	"yaagwadb"

	"github.com/pkg/errors"
)

const (
	decimalBase          = 10
	integerBitSize       = 64
	valueErrorFormatType = "%w: value of type %q"
	valueErrorFormatBlob = "%w: BLOB values are not supported; pass file data as a base64 string"
	valueErrorFormatText = "%w: %v"
)

func toQueryValue(input value) (yaagwadb.QueryValue, error) {
	switch input.Type {
	case valueTypeNull:
		return yaagwadb.QueryValue{}, nil
	case valueTypeInteger:
		var text string
		if err := json.Unmarshal(input.Value, &text); err != nil {
			return yaagwadb.QueryValue{}, fmt.Errorf(valueErrorFormatText, errors.New(yaagwadb.InvalidArgument), err)
		}
		integer, err := strconv.ParseInt(text, decimalBase, integerBitSize)
		if err != nil {
			return yaagwadb.QueryValue{}, fmt.Errorf(valueErrorFormatText, errors.New(yaagwadb.InvalidArgument), err)
		}
		return yaagwadb.QueryValue{Integer: &integer}, nil
	case valueTypeFloat:
		var real float64
		if err := json.Unmarshal(input.Value, &real); err != nil {
			return yaagwadb.QueryValue{}, fmt.Errorf(valueErrorFormatText, errors.New(yaagwadb.InvalidArgument), err)
		}
		return yaagwadb.QueryValue{Real: &real}, nil
	case valueTypeText:
		var text string
		if err := json.Unmarshal(input.Value, &text); err != nil {
			return yaagwadb.QueryValue{}, fmt.Errorf(valueErrorFormatText, errors.New(yaagwadb.InvalidArgument), err)
		}
		return yaagwadb.QueryValue{Text: &text}, nil
	case valueTypeBlob:
		return yaagwadb.QueryValue{}, fmt.Errorf(valueErrorFormatBlob, errors.New(TypeMismatch))
	}
	return yaagwadb.QueryValue{}, fmt.Errorf(valueErrorFormatType, errors.New(TypeMismatch), input.Type)
}

func fromQueryValue(input yaagwadb.QueryValue) value {
	switch {
	case input.Integer != nil:
		encoded, _ := json.Marshal(strconv.FormatInt(*input.Integer, decimalBase))
		return value{Type: valueTypeInteger, Value: encoded}
	case input.Real != nil:
		encoded, _ := json.Marshal(*input.Real)
		return value{Type: valueTypeFloat, Value: encoded}
	case input.Text != nil:
		return textValue(*input.Text)
	}
	return value{Type: valueTypeNull}
}

func textValue(text string) value {
	encoded, _ := json.Marshal(text)
	return value{Type: valueTypeText, Value: encoded}
}

// arguments holds the parameters of one statement: positional ones by their number (from 1) and named ones by name with prefix.
type arguments struct {
	positional []yaagwadb.QueryValue
	named      map[string]yaagwadb.QueryValue
}

func newArguments(input *statement) (*arguments, error) {
	result := &arguments{named: map[string]yaagwadb.QueryValue{}}
	for _, argument := range input.Args {
		converted, err := toQueryValue(argument)
		if err != nil {
			return nil, err
		}
		result.positional = append(result.positional, converted)
	}
	for _, argument := range input.NamedArgs {
		converted, err := toQueryValue(argument.Value)
		if err != nil {
			return nil, err
		}
		result.named[argument.Name] = converted
	}
	return result, nil
}

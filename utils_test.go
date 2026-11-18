package yaagwadb

import (
	"fmt"
	"strings"
	"syscall"
	"testing"

	"github.com/pkg/errors"
)

const (
	// Format of the detail attached to NotFound
	testInodeDetailFormat = "%w: inode 3"
	// Format for wrapping Corrupted
	testWrapFormat = "read: %w"
	// Format for wrapping ENOSPC
	testWriteWrapFormat = "write: %w"
	// Detail appended after Exists
	testExistsDetail = ": \"a\""
	// Function and file names the stack must contain
	testStackFunctionName = "TestStackOf"
	testStackFileName     = "utils_test.go"
	// Message of an error without a stack
	testPlainErrorMessage = "plain"
	// Failure message format when Has returns the wrong result
	testHasFormat = "Has(%v)"
	// Failure message when Has(nil) returns true
	testHasNilMessage = "Has(nil)"
	// Failure message format when messageOf returns the wrong result
	testMessageOfFormat = "messageOf(%v) = %q, want %q"
	// Failure message format when the stack does not point to the line that created the error
	testStackFormat = "stack does not point to the line that created the error:\n%s"
	// Failure message when Has does not match the constant after wrapping
	testHasConstantMessage = "Has must still match the constant"
	// Failure message when a stack is returned for an error without one
	testErrorsWithoutStackMessage = "errors without a stack"
)

func TestHas(t *testing.T) {
	err := fmt.Errorf(testInodeDetailFormat, errors.New(NotFound))
	if !Has(err, NotFound) || Has(err, Exists) {
		t.Fatalf(testHasFormat, err)
	}
	if Has(nil, NotFound) {
		t.Fatal(testHasNilMessage)
	}
}

func TestMessageOf(t *testing.T) {
	cases := map[string]error{
		Corrupted:       fmt.Errorf(testWrapFormat, errors.New(Corrupted)),
		NoSpace:         fmt.Errorf(testWriteWrapFormat, syscall.ENOSPC),
		Exists:          errors.New(Exists + testExistsDetail),
		testEmptyString: errors.New(testOtherErrorMessage),
	}
	for want, err := range cases {
		if got := messageOf(err); got != want {
			t.Errorf(testMessageOfFormat, err, got, want)
		}
	}
}

func TestStackOf(t *testing.T) {
	err := fmt.Errorf(testErrorDetailFormat, errors.New(NotFound))
	stack := StackOf(err)
	if !strings.Contains(stack, testStackFunctionName) || !strings.Contains(stack, testStackFileName) {
		t.Fatalf(testStackFormat, stack)
	}
	if !Has(err, NotFound) {
		t.Fatal(testHasConstantMessage)
	}
	if StackOf(fmt.Errorf(testPlainErrorMessage)) != testEmptyString || StackOf(nil) != testEmptyString {
		t.Fatal(testErrorsWithoutStackMessage)
	}
}

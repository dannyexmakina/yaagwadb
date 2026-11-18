package yaagwadb

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

const (
	// stackTraceFormat formats a stack with both function names and file:line.
	stackTraceFormat = "%+v"
	// noMessage means the error chain has no constant message or stack.
	noMessage = ""
)

func Has(err error, message string) bool {
	for err != nil {
		if err.Error() == message {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

// stackTracer extracts the stack from errors returned by github.com/pkg/errors.
type stackTracer interface {
	StackTrace() errors.StackTrace
}

func StackOf(err error) string {
	var stack errors.StackTrace
	for ; err != nil; err = errors.Unwrap(err) {
		if tracer, ok := err.(stackTracer); ok {
			stack = tracer.StackTrace()
		}
	}
	if stack == nil {
		return noMessage
	}
	return fmt.Sprintf(stackTraceFormat, stack)
}

func messageOf(err error) string {
	if err == nil {
		return noMessage
	}
	for _, candidate := range errorMessages {
		if Has(err, candidate) {
			return candidate
		}
	}
	if errors.Is(err, syscall.ENOSPC) {
		return NoSpace
	}
	// Errors turned back into strings, such as a task's LastError, are matched by prefix
	for _, candidate := range errorMessages {
		if strings.HasPrefix(err.Error(), candidate) {
			return candidate
		}
	}
	return noMessage
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

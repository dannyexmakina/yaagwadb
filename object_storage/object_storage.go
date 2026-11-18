package objectstorage

import (
	"context"
	"io"
	"iter"
)

const (
	NotFound        = "object storage: not found"
	InvalidArgument = "object storage: invalid argument"
)

const (
	// notFoundFormat attaches the key to NotFound.
	notFoundFormat = "%w: %s"
	// sizeMismatchFormat reports a mismatch between the written and the specified size.
	sizeMismatchFormat = "%w: size mismatch %d != %d"
	// minimumKnownSize is the smallest valid Put size. Smaller sizes mean "unknown" and are not checked.
	minimumKnownSize = 0
)

// ObjectStorage is the minimal set of object storage operations.
type ObjectStorage interface {
	Put(ctx context.Context, key string, r io.Reader, size int64) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Head(ctx context.Context, key string) (size int64, err error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) iter.Seq2[string, error]
}

func Has(err error, message string) bool {
	for err != nil {
		if err.Error() == message {
			return true
		}
		type unwrapper interface{ Unwrap() error }
		next, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = next.Unwrap()
	}
	return false
}

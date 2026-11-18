package objectstorage

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"strings"
	"testing"
)

const (
	// testBucketNameFormat formats a bucket name unique to each S3/GCS test.
	testBucketNameFormat = "yaagwadb-test-%d"
	// exerciseListPrefix is the prefix List filters by.
	exerciseListPrefix = "p/"
	// exerciseKeyFirstUnderPrefix and exerciseKeySecondUnderPrefix are keys under exerciseListPrefix (a sorts first).
	exerciseKeyFirstUnderPrefix  = exerciseListPrefix + "a"
	exerciseKeySecondUnderPrefix = exerciseListPrefix + "b"
	// exerciseKeyOtherPrefix is a key with another prefix that must not appear in List results.
	exerciseKeyOtherPrefix = "q/c"
	// exerciseListSeparator and exerciseExpectedListing form the expected joined List result.
	exerciseListSeparator   = ","
	exerciseExpectedListing = exerciseKeyFirstUnderPrefix + exerciseListSeparator + exerciseKeySecondUnderPrefix
	// exerciseReplacedFileData is the content used for overwriting.
	exerciseReplacedFileData = "replaced"
	// exerciseLargeObjectSize is large enough to split into multiple parts (9 MiB).
	exerciseLargeObjectSize = 9 << 20
	// exerciseLargeObjectKey is the key of the large object.
	exerciseLargeObjectKey = "filedata/0123456789abcdef0123456789abcdef"
	// Failure messages.
	exercisePutFailureFormat              = "put %s: %v"
	exerciseGetFailureFormat              = "get: %q"
	exerciseHeadFailureFormat             = "head: %d %v"
	exerciseOverwriteFailureFormat        = "overwrite: size %d"
	exerciseLargeObjectMismatch           = "large object mismatch"
	exerciseListFailureFormat             = "list: %v"
	exerciseDeleteIdempotentFailureFormat = "delete should be idempotent: %v"
	exerciseGetAfterDeleteFailureFormat   = "get after delete: want NotFound, got %v"
	exerciseHeadAfterDeleteFailureFormat  = "head after delete: want NotFound, got %v"
	// hasTestOuterMessage and hasTestMismatch are values for TestHas.
	hasTestOuterMessage = "outer"
	hasTestMismatch     = "Has mismatch"
)

func exerciseObjectStorage(t *testing.T, storage ObjectStorage) {
	t.Helper()
	ctx := context.Background()
	for _, key := range []string{exerciseKeySecondUnderPrefix, exerciseKeyFirstUnderPrefix, exerciseKeyOtherPrefix} {
		if err := storage.Put(ctx, key, strings.NewReader(key), int64(len(key))); err != nil {
			t.Fatalf(exercisePutFailureFormat, key, err)
		}
	}
	reader, err := storage.Get(ctx, exerciseKeyFirstUnderPrefix)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(reader)
	reader.Close()
	if string(data) != exerciseKeyFirstUnderPrefix {
		t.Fatalf(exerciseGetFailureFormat, data)
	}
	if size, err := storage.Head(ctx, exerciseKeyOtherPrefix); err != nil || size != int64(len(exerciseKeyOtherPrefix)) {
		t.Fatalf(exerciseHeadFailureFormat, size, err)
	}

	// Overwrite
	if err := storage.Put(ctx, exerciseKeyFirstUnderPrefix, strings.NewReader(exerciseReplacedFileData), int64(len(exerciseReplacedFileData))); err != nil {
		t.Fatal(err)
	}
	if size, _ := storage.Head(ctx, exerciseKeyFirstUnderPrefix); size != int64(len(exerciseReplacedFileData)) {
		t.Fatalf(exerciseOverwriteFailureFormat, size)
	}

	// Object the size of a file data (large enough for multiple parts)
	large := make([]byte, exerciseLargeObjectSize)
	if _, err := rand.Read(large); err != nil {
		t.Fatal(err)
	}
	if err := storage.Put(ctx, exerciseLargeObjectKey, bytes.NewReader(large), int64(len(large))); err != nil {
		t.Fatal(err)
	}
	reader, err = storage.Get(ctx, exerciseLargeObjectKey)
	if err != nil {
		t.Fatal(err)
	}
	data, _ = io.ReadAll(reader)
	reader.Close()
	if !bytes.Equal(data, large) {
		t.Fatal(exerciseLargeObjectMismatch)
	}

	var keys []string
	for key, err := range storage.List(ctx, exerciseListPrefix) {
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, key)
	}
	if strings.Join(keys, exerciseListSeparator) != exerciseExpectedListing {
		t.Fatalf(exerciseListFailureFormat, keys)
	}

	if err := storage.Delete(ctx, exerciseKeyFirstUnderPrefix); err != nil {
		t.Fatal(err)
	}
	if err := storage.Delete(ctx, exerciseKeyFirstUnderPrefix); err != nil {
		t.Fatalf(exerciseDeleteIdempotentFailureFormat, err)
	}
	if _, err := storage.Get(ctx, exerciseKeyFirstUnderPrefix); !Has(err, NotFound) {
		t.Fatalf(exerciseGetAfterDeleteFailureFormat, err)
	}
	if _, err := storage.Head(ctx, exerciseKeyFirstUnderPrefix); !Has(err, NotFound) {
		t.Fatalf(exerciseHeadAfterDeleteFailureFormat, err)
	}
}

func TestHas(t *testing.T) {
	if Has(io.EOF, NotFound) || !Has(&wrapped{inner: NotFound}, NotFound) || Has(nil, NotFound) {
		t.Fatal(hasTestMismatch)
	}
}

type wrapped struct{ inner string }

func (w *wrapped) Error() string { return hasTestOuterMessage }
func (w *wrapped) Unwrap() error { return errorString(w.inner) }

type errorString string

func (e errorString) Error() string { return string(e) }

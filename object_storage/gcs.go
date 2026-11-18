package objectstorage

import (
	"context"
	"fmt"
	"io"
	"iter"

	"github.com/pkg/errors"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// GoogleCloudStorage is the Google Cloud Storage implementation.
type GoogleCloudStorage struct {
	bucket *storage.BucketHandle
}

func NewGoogleCloudStorage(client *storage.Client, bucket string) *GoogleCloudStorage {
	return &GoogleCloudStorage{bucket: client.Bucket(bucket)}
}

func googleCloudStorageNotFound(key string, err error) error {
	if errors.Is(err, storage.ErrObjectNotExist) {
		return fmt.Errorf(notFoundFormat, errors.New(NotFound), key)
	}
	return err
}

func (g *GoogleCloudStorage) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	writer := g.bucket.Object(key).NewWriter(ctx)
	written, err := io.Copy(writer, r)
	if err != nil {
		writer.Close()
		return err
	}
	if size >= minimumKnownSize && written != size {
		writer.Close()
		return fmt.Errorf(sizeMismatchFormat, errors.New(InvalidArgument), written, size)
	}
	return writer.Close()
}

func (g *GoogleCloudStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	reader, err := g.bucket.Object(key).NewReader(ctx)
	if err != nil {
		return nil, googleCloudStorageNotFound(key, err)
	}
	return reader, nil
}

func (g *GoogleCloudStorage) Head(ctx context.Context, key string) (int64, error) {
	attributes, err := g.bucket.Object(key).Attrs(ctx)
	if err != nil {
		return 0, googleCloudStorageNotFound(key, err)
	}
	return attributes.Size, nil
}

func (g *GoogleCloudStorage) Delete(ctx context.Context, key string) error {
	err := g.bucket.Object(key).Delete(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return nil
	}
	return err
}

func (g *GoogleCloudStorage) List(ctx context.Context, prefix string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		objects := g.bucket.Objects(ctx, &storage.Query{Prefix: prefix})
		for {
			attributes, err := objects.Next()
			if errors.Is(err, iterator.Done) {
				return
			}
			if err != nil {
				yield("", err)
				return
			}
			if !yield(attributes.Name, nil) {
				return
			}
		}
	}
}

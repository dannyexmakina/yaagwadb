package objectstorage

import (
	"context"
	"fmt"
	"io"
	"iter"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

const (
	s3NotFoundErrorCode  = "NotFound"
	s3NoSuchKeyErrorCode = "NoSuchKey"
)

// S3 is the implementation for S3-compatible storage.
type S3 struct {
	client *s3.Client
	bucket string
}

func NewS3(client *s3.Client, bucket string) *S3 {
	return &S3{client: client, bucket: bucket}
}

func s3NotFound(key string, err error) error {
	var noSuchKey *types.NoSuchKey
	var notFoundType *types.NotFound
	var apiError smithy.APIError
	if errors.As(err, &noSuchKey) || errors.As(err, &notFoundType) ||
		(errors.As(err, &apiError) && (apiError.ErrorCode() == s3NotFoundErrorCode || apiError.ErrorCode() == s3NoSuchKeyErrorCode)) {
		return fmt.Errorf(notFoundFormat, errors.New(NotFound), key)
	}
	return err
}

func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), Body: r, ContentLength: aws.Int64(size),
	})
	return err
}

func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, s3NotFound(key, err)
	}
	return output.Body, nil
}

func (s *S3) Head(ctx context.Context, key string) (int64, error) {
	output, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return 0, s3NotFound(key, err)
	}
	return aws.ToInt64(output.ContentLength), nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	return err
}

func (s *S3) List(ctx context.Context, prefix string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
			Bucket: aws.String(s.bucket), Prefix: aws.String(prefix),
		})
		for paginator.HasMorePages() {
			page, err := paginator.NextPage(ctx)
			if err != nil {
				yield("", err)
				return
			}
			for _, object := range page.Contents {
				if !yield(aws.ToString(object.Key), nil) {
					return
				}
			}
		}
	}
}

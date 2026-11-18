package objectstorage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	// s3TestEndpointVariable is the environment variable giving the S3-compatible container endpoint.
	s3TestEndpointVariable = "YAAGWADB_S3_ENDPOINT"
	// s3TestUnsetEndpoint means no endpoint is set.
	s3TestUnsetEndpoint = ""
	// s3TestSkipMessage is the skip reason when no endpoint is set.
	s3TestSkipMessage = "set YAAGWADB_S3_ENDPOINT (see yaagwadb-test develop/compose.yaml)"
	// s3TestRegion is the signing region (arbitrary on the compatible container).
	s3TestRegion = "us-east-1"
	// s3TestAccessKey and s3TestSecretKey are the credentials set on the yaagwadb-test develop/compose.yaml container.
	s3TestAccessKey = "yaagwadb"
	s3TestSecretKey = "yaagwadb-secret"
	// s3TestNoSessionToken means no temporary credentials are used.
	s3TestNoSessionToken = ""
	// s3TestUsePathStyle uses path-style buckets because the compatible container does not support virtual-hosted style.
	s3TestUsePathStyle = true
)

func TestS3(t *testing.T) {
	endpoint := os.Getenv(s3TestEndpointVariable)
	if endpoint == s3TestUnsetEndpoint {
		t.Skip(s3TestSkipMessage)
	}
	ctx := context.Background()
	client := s3.NewFromConfig(aws.Config{
		Region:      s3TestRegion,
		Credentials: credentials.NewStaticCredentialsProvider(s3TestAccessKey, s3TestSecretKey, s3TestNoSessionToken),
	}, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = s3TestUsePathStyle
	})
	bucket := fmt.Sprintf(testBucketNameFormat, time.Now().UnixNano())
	if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
		t.Fatal(err)
	}
	exerciseObjectStorage(t, NewS3(client, bucket))
}

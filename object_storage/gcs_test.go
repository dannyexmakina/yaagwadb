package objectstorage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/storage"
)

const (
	// googleCloudStorageTestEndpointVariable is the environment variable giving the emulator endpoint.
	googleCloudStorageTestEndpointVariable = "YAAGWADB_GCS_ENDPOINT"
	// googleCloudStorageTestUnsetEndpoint means no endpoint is set.
	googleCloudStorageTestUnsetEndpoint = ""
	// googleCloudStorageTestSkipMessage is the skip reason when no endpoint is set.
	googleCloudStorageTestSkipMessage = "set YAAGWADB_GCS_ENDPOINT (see file-db-test develop/compose.yaml)"
	// googleCloudStorageTestEmulatorVariable is the environment variable that points the client library at the emulator.
	googleCloudStorageTestEmulatorVariable = "STORAGE_EMULATOR_HOST"
	// googleCloudStorageTestProject is the project for creating buckets (arbitrary on the emulator).
	googleCloudStorageTestProject = "yaagwadb-test"
)

func TestGoogleCloudStorage(t *testing.T) {
	endpoint := os.Getenv(googleCloudStorageTestEndpointVariable)
	if endpoint == googleCloudStorageTestUnsetEndpoint {
		t.Skip(googleCloudStorageTestSkipMessage)
	}
	// With this variable set, the client library targets the emulator and skips authentication
	t.Setenv(googleCloudStorageTestEmulatorVariable, endpoint)
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	bucket := fmt.Sprintf(testBucketNameFormat, time.Now().UnixNano())
	if err := client.Bucket(bucket).Create(ctx, googleCloudStorageTestProject, nil); err != nil {
		t.Fatal(err)
	}
	exerciseObjectStorage(t, NewGoogleCloudStorage(client, bucket))
}

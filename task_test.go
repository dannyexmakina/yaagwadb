package yaagwadb

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

const (
	// Display names of task kinds
	testTaskKindDeleteFileData = "DeleteFileData"
	// Display name of the index archive PUT (matches task.go)
	testTaskKindPutIndexName = "PutIndexArchive"
	// Number of a task kind that matches no kind, and its display
	testUnknownTaskKind     = 9
	testUnknownTaskKindName = "TaskKind(9)"
	// Retry for an always-failing object storage write (long enough not to rerun during the test)
	testStalledRetryWait        = time.Hour
	testStalledRetryMaxAttempts = 100
	testStalledRetryMaxDuration = 24 * time.Hour
	// Number of tasks ListTasks fetches
	testPendingTasksLimit = 10
	// Message format when the object storage tasks differ from the expectation
	testObjectStorageTasksFormat = "object storage tasks %+v"
	// Failure message format when a task kind display name differs from the expectation
	testTaskKindNameWantFormat = "got %s, want %s"
	// Failure message format when a task kind display name is wrong
	testTaskKindNameFormat = "got %s"
)

func TestTaskKindName(t *testing.T) {
	if got := TaskKindName(FileDataDeleteQueue, TaskDeleteFileData); got != testTaskKindDeleteFileData {
		t.Errorf(testTaskKindNameWantFormat, got, testTaskKindDeleteFileData)
	}
	if got := TaskKindName(ObjectStorageWriteQueue, TaskPutIndexArchive); got != testTaskKindPutIndexName {
		t.Errorf(testTaskKindNameFormat, got)
	}
	if got := TaskKindName(FileDataDeleteQueue, testUnknownTaskKind); got != testUnknownTaskKindName {
		t.Errorf(testTaskKindNameFormat, got)
	}
}

func TestQueuesAreIndependent(t *testing.T) {
	config := testConfig(t.TempDir())
	config.ObjectStorageWriteRetry = RetryPolicy{Base: testStalledRetryWait, Cap: testStalledRetryWait, MaxAttempts: testStalledRetryMaxAttempts, MaxDuration: testStalledRetryMaxDuration}
	daemon := openWithFailingStorage(t, config, &failingObjectStorage{})
	if _, err := daemon.Write(RootInodeNumber, testFileName, testFileMode, testNoOwnerID, testNoOwnerID, bytes.NewReader([]byte(testFileData))); err != nil {
		t.Fatal(err)
	}
	if _, err := daemon.Lookup(RootInodeNumber, testFileName); err != nil {
		t.Fatal(err)
	}
	// The object storage write fails and stays waiting for retry
	eventually(t, func() error {
		tasks, err := daemon.ListTasks(ObjectStorageWriteQueue, testListActiveTasksOnly, testPendingTasksLimit)
		if err != nil {
			return err
		}
		if len(tasks) != testExpectedSingle || tasks[testFirstElementIndex].State != TaskPending || tasks[testFirstElementIndex].LastError == testEmptyString {
			return fmt.Errorf(testObjectStorageTasksFormat, tasks)
		}
		return nil
	})
}

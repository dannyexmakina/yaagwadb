package yaagwadb

import (
	"context"
	"database/sql"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/pkg/errors"

	objectstorage "yaagwadb/object_storage"
)

const (
	// testRunningTaskTimeout is the running task timeout of the recovery test; the task is made older by testRunningTaskAgeFactor
	testRunningTaskTimeout   = time.Minute
	testRunningTaskAgeFactor = 2
	// Duration that makes grace and retention periods expire immediately
	testElapsedImmediately = time.Nanosecond
	// Grace period that does not expire during the test
	testLongGracePeriod = time.Hour
	// Index backup interval used with object storage when the grace period must expire immediately (the scheduler is disabled, so it never runs)
	testIndexArchiveIntervalImmediately = time.Nanosecond
	// Grace period that expires immediately but is still longer than testIndexArchiveIntervalImmediately
	testGracePeriodImmediately = 2 * time.Nanosecond
	// Expected number of failed tasks
	testExpectedFailedTasks = 1
	// Without writes, the index archive is queued only once
	testExpectedIndexArchiveCount = 1
	// Number of tasks ListTasks fetches (for checking object storage writes)
	testObjectStorageTasksLimit = 10
	// Message when a file data entry remains after physical deletion
	testFileDataEntryRemainsMessage = "file data entry remains"
	// Failure message format when an object remains in object storage
	testObjectRemainsFormat = "object remains: %v"
	// Failure message format when a file data disappears during the grace period
	testGracePeriodFileDataFormat = "file data must stay during the grace period: %v"
	// Failure message format when the index archive is queued more than once
	testArchiveCountFormat = "no write since the last archive, but %d archives"
)

func TestPhysicalDelete(t *testing.T) {
	storage := newMemoryObjectStorage(t)
	config := testConfig(t.TempDir())
	config.ObjectStorage = storage
	config.IndexArchiveInterval = testIndexArchiveIntervalImmediately
	config.LogicalDeleteGracePeriod = testGracePeriodImmediately
	daemon := openTestDaemon(t, config)
	inode := writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testFileData))
	if err := daemon.Unlink(RootInodeNumber, testFileName); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	if err := daemon.schedulePhysicalDelete(); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)

	if err := daemon.index.view(func(transaction *sql.Tx) error {
		exists, err := hasFileData(transaction, inode.FileDataID)
		if err == nil && exists {
			return errors.New(testFileDataEntryRemainsMessage)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(FileDataPath(daemon.fileDataDirectory, inode.FileDataID)); !os.IsNotExist(err) {
		t.Fatalf(testFileDataRemainsFormat, err)
	}
	if _, err := storage.Head(context.Background(), testObjectFileDataPrefix+inode.FileDataID); !objectstorage.Has(err, objectstorage.NotFound) {
		t.Fatalf(testObjectRemainsFormat, err)
	}
}

func TestPhysicalDeleteWaitsForGracePeriod(t *testing.T) {
	config := testConfig(t.TempDir())
	config.LogicalDeleteGracePeriod = testLongGracePeriod
	daemon := openTestDaemon(t, config)
	inode := writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testFileData))
	if err := daemon.Unlink(RootInodeNumber, testFileName); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	if err := daemon.RunScheduledJobs(); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	if _, err := os.Stat(FileDataPath(daemon.fileDataDirectory, inode.FileDataID)); err != nil {
		t.Fatalf(testGracePeriodFileDataFormat, err)
	}
}

func TestRetryFailedTasks(t *testing.T) {
	config := testConfig(t.TempDir())
	storage := &failingObjectStorage{}
	daemon := openWithFailingStorage(t, config, storage)
	writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testFileData))
	status, err := daemon.Status()
	if err != nil || status.Queues[slices.Index(queueNames, ObjectStorageWriteQueue)].Failed != testExpectedFailedTasks {
		t.Fatalf(testStatusFormat, status, err)
	}
	storage.recovered.Store(newMemoryObjectStorage(t))
	if err := daemon.retryFailedTasks(); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	tasks, err := daemon.ListTasks(ObjectStorageWriteQueue, testListAllTasks, testObjectStorageTasksLimit)
	if err != nil || len(tasks) != testExpectedSingle || tasks[testFirstElementIndex].State != TaskSucceeded {
		t.Fatalf(testTasksFormat, tasks, err)
	}
}

func TestScheduleIndexArchive(t *testing.T) {
	config := testConfig(t.TempDir())
	config.ObjectStorage = newMemoryObjectStorage(t)
	daemon := openTestDaemon(t, config)
	writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testFileData))
	count := func() int {
		tasks, err := daemon.ListTasks(ObjectStorageWriteQueue, testListAllTasks, testListTasksLimit)
		if err != nil {
			t.Fatal(err)
		}
		archives := testZeroCount
		for _, task := range tasks {
			if task.Kind == TaskPutIndexArchive {
				archives++
			}
		}
		return archives
	}
	if err := daemon.scheduleIndexArchive(); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	if err := daemon.scheduleIndexArchive(); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	if got := count(); got != testExpectedIndexArchiveCount {
		t.Fatalf(testArchiveCountFormat, got)
	}
}

func TestRemoveFinishedTasks(t *testing.T) {
	config := testConfig(t.TempDir())
	config.FinishedTaskRetention = testElapsedImmediately
	daemon := openTestDaemon(t, config)
	writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testFileData))
	if err := daemon.removeFinishedTasks(); err != nil {
		t.Fatal(err)
	}
	tasks, err := daemon.ListTasks(allTaskQueues, testListAllTasks, testListTasksLimit)
	if err != nil || len(tasks) != testZeroCount {
		t.Fatalf(testTasksFormat, tasks, err)
	}
}

func TestRecoverRunningTasks(t *testing.T) {
	config := testConfig(t.TempDir())
	config.RunningTaskTimeout = testRunningTaskTimeout
	daemon := openTestDaemon(t, config)
	var taskID uint64
	// A task a worker took and then stopped on: running and started long ago
	if err := daemon.index.update(func(transaction *sql.Tx) error {
		task, err := enqueueTask(transaction, FileDataDeleteQueue, fileDataDeleteTask(newFileDataID()))
		if err != nil {
			return err
		}
		taskID = task.ID
		task.State = TaskRunning
		task.StartedAt = time.Now().Add(-testRunningTaskTimeout * testRunningTaskAgeFactor).UnixNano()
		return putTask(transaction, FileDataDeleteQueue, task)
	}); err != nil {
		t.Fatal(err)
	}
	if err := daemon.recoverRunningTasks(); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	if err := daemon.index.view(func(transaction *sql.Tx) error {
		task, err := getTask(transaction, FileDataDeleteQueue, taskID)
		if err == nil && task.State != TaskSucceeded {
			return errors.New(task.State.String())
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

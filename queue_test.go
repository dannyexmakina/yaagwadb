package yaagwadb

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/pkg/errors"
)

const (
	// Initial wait and cap used to compute wait times
	testDelayBase = time.Second
	testDelayCap  = 10 * time.Second
	// Attempt numbers for checking wait times (1st, 2nd, 4th, and one past the cap)
	testAttemptFirst     = 1
	testAttemptSecond    = 2
	testAttemptFourth    = 4
	testAttemptBeyondCap = 10
	// Key and payload of the task enqueued in the unique key test
	testUniqueTaskKey     = "k"
	testUniqueTaskPayload = "p"

	// Prefix of file names written by the child process
	testCrashFileNamePrefix = "file"
	// Number of file names reused by the child process
	testCrashFileNameCount = 20
	// Number of writes between Unlinks
	testCrashUnlinkEvery = 3
	// Offset of the Unlinked name from the written name
	testCrashUnlinkOffset = 7
	// Number of writes before the child process signals readiness
	testCrashReadyWriteCount = 10
	// testCrashWriteIncrement is the amount counted per completed write.
	testCrashWriteIncrement = 1
	// Cycle of the content padding length
	testCrashPaddingCycle = 5000
	// File data padding character
	testCrashPaddingCharacter = "x"
	// File data format (name:sequence:padding)
	testCrashFileDataFormat = "%s:%d:%s"
	// Separator after the name in the content
	testCrashFileDataSeparator = ":"
	// Number of kill -9 rounds
	testCrashRounds = 3
	// Per-round increment of the kill -9 delay
	testCrashKillDelayStep = 20 * time.Millisecond
	// Arguments to run only TestCrashChild as the child process
	testCrashChildRunArgument = "-test.run=^TestCrashChild$"
	// Argument for verbose child output (so READY reaches stdout)
	testVerboseArgument = "-test.v"
	// Separator between an environment variable name and value
	testEnvironmentSeparator = "="
	// Skip reason under -short
	testSkipShortReason = "short"
	// Skip reason when not running as the child process
	testSkipNotChildReason = "child process of TestCrashProcessKill"

	// Error message and format returned on unexpected results
	testFirstEnqueueFormat         = "first %v %v"
	testDuplicateQueuedFormat      = "duplicate was queued: %v %v"
	testKeyRemainsMessage          = "key remains after finish"
	testThirdEnqueueFormat         = "third %v %v"
	testUnregisteredFileDataFormat = "unregistered file data %s"
	// Failure message format for an unexpected delay result
	testDelayFormat = "delay(%d) = %v, want %v"
	// Failure message format when a file has wrong content after kill -9
	testRoundWrongContentFormat = "round %d: %s has wrong content (%d bytes)"
	// Failure message format when state is inconsistent after kill -9
	testRoundErrorFormat = "round %d: %v"
	// Remainder meaning evenly divisible
	testNoRemainder = 0
)

func TestRetryPolicyDelay(t *testing.T) {
	policy := RetryPolicy{Base: testDelayBase, Cap: testDelayCap}
	for attempts, want := range map[uint32]time.Duration{
		testAttemptFirst:     testDelayBase,
		testAttemptSecond:    testDelayBase << (testAttemptSecond - testAttemptFirst),
		testAttemptFourth:    testDelayBase << (testAttemptFourth - testAttemptFirst),
		testAttemptBeyondCap: testDelayCap,
	} {
		if got := policy.delay(attempts); got != want {
			t.Errorf(testDelayFormat, attempts, got, want)
		}
	}
}

func TestEnqueueUniqueAndKeyRelease(t *testing.T) {
	daemon := openTestDaemon(t, testConfig(t.TempDir()))
	err := daemon.index.update(func(transaction *sql.Tx) error {
		request := TaskRequest{Kind: TaskDeleteFileData, Key: testUniqueTaskKey, Payload: []byte(testUniqueTaskPayload)}
		first, err := enqueueTask(transaction, FileDataDeleteQueue, request)
		if err != nil || first == nil {
			return fmt.Errorf(testFirstEnqueueFormat, first, err)
		}
		if second, err := enqueueTask(transaction, FileDataDeleteQueue, request); err != nil || second != nil {
			return fmt.Errorf(testDuplicateQueuedFormat, second, err)
		}
		if err := finishTask(transaction, FileDataDeleteQueue, first, TaskSucceeded, testEmptyString); err != nil {
			return err
		}
		if exists, err := hasActiveTask(transaction, FileDataDeleteQueue, testUniqueTaskKey); err != nil || exists {
			return errors.New(testKeyRemainsMessage)
		}
		if third, err := enqueueTask(transaction, FileDataDeleteQueue, request); err != nil || third == nil {
			return fmt.Errorf(testThirdEnqueueFormat, third, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestResetRunningTasks(t *testing.T) {
	directory := t.TempDir()
	config := testConfig(directory)
	daemon, err := Open(config)
	if err != nil {
		t.Fatal(err)
	}
	// Create a Running task with the workers stopped (simulating a crash mid-processing)
	daemon.Close()
	tasks, err := openDatabase(filepath.Join(directory, defaultDatabaseFileName), testDatabaseBusyTimeout)
	if err != nil {
		t.Fatal(err)
	}
	fileDataID := newFileDataID()
	var taskID uint64
	if err := tasks.update(func(transaction *sql.Tx) error {
		task, err := enqueueTask(transaction, FileDataDeleteQueue, fileDataDeleteTask(fileDataID))
		if err != nil {
			return err
		}
		taskID = task.ID
		task.State = TaskRunning
		task.StartedAt = time.Now().UnixNano()
		return putTask(transaction, FileDataDeleteQueue, task)
	}); err != nil {
		t.Fatal(err)
	}
	tasks.close()

	// On restart it is reset to Pending and a worker retries it
	daemon = openTestDaemon(t, config)
	waitIdle(t, daemon)
	if err := daemon.index.view(func(transaction *sql.Tx) error {
		task, err := getTask(transaction, FileDataDeleteQueue, taskID)
		if err != nil {
			return err
		}
		if task.State != TaskSucceeded {
			return errors.New(task.State.String())
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

const (
	// Name of the environment variable passing the data directory to the child process
	crashChildDirectory = "YAAGWADB_CRASH_DIRECTORY"
	// Line the child process prints when it starts writing
	crashReady = "READY"
)

func TestCrashChild(t *testing.T) {
	directory := os.Getenv(crashChildDirectory)
	if directory == testEmptyString {
		t.Skip(testSkipNotChildReason)
	}
	daemon, err := Open(testConfig(directory))
	if err != nil {
		t.Fatal(err)
	}
	var count atomic.Int64
	for index := testFirstElementIndex; ; index++ {
		name := testCrashFileNamePrefix + strconv.Itoa(index%testCrashFileNameCount)
		if _, err := daemon.Write(RootInodeNumber, name, testFileMode, testNoOwnerID, testNoOwnerID, bytes.NewReader(crashFileData(name, index))); err != nil {
			t.Fatal(err)
		}
		if index%testCrashUnlinkEvery == testNoRemainder {
			_ = daemon.Unlink(RootInodeNumber, testCrashFileNamePrefix+strconv.Itoa((index+testCrashUnlinkOffset)%testCrashFileNameCount))
		}
		if count.Add(testCrashWriteIncrement) == testCrashReadyWriteCount {
			fmt.Println(crashReady)
		}
	}
}

func crashFileData(name string, index int) []byte {
	return []byte(fmt.Sprintf(testCrashFileDataFormat, name, index, bytes.Repeat([]byte(testCrashPaddingCharacter), index%testCrashPaddingCycle)))
}

func TestCrashProcessKill(t *testing.T) {
	if testing.Short() {
		t.Skip(testSkipShortReason)
	}
	for round := range testCrashRounds {
		directory := t.TempDir()
		command := exec.Command(os.Args[testFirstElementIndex], testCrashChildRunArgument, testVerboseArgument)
		command.Env = append(os.Environ(), crashChildDirectory+testEnvironmentSeparator+directory)
		output, err := command.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(output)
		for scanner.Scan() && scanner.Text() != crashReady {
		}
		time.Sleep(time.Duration(round) * testCrashKillDelayStep)
		command.Process.Signal(syscall.SIGKILL)
		command.Wait()

		daemon := openTestDaemon(t, testConfig(directory))
		waitIdle(t, daemon)
		entries, err := daemon.ListDirectory(RootInodeNumber)
		if err != nil {
			t.Fatal(err)
		}
		referenced := map[string]bool{}
		for _, entry := range entries {
			data := readAll(t, daemon, entry.Inode.InodeNumber)
			if !bytes.HasPrefix(data, []byte(entry.Name+testCrashFileDataSeparator)) || uint64(len(data)) != entry.Inode.Size {
				t.Fatalf(testRoundWrongContentFormat, round, entry.Name, len(data))
			}
		}
		if err := daemon.index.view(func(transaction *sql.Tx) error {
			var err error
			referenced, err = listFileDataIDs(transaction)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		// Startup cleanup has removed file data not in the index
		if err := filepath.WalkDir(daemon.fileDataDirectory, func(path string, entry fs.DirEntry, err error) error {
			if err == nil && !entry.IsDir() && !referenced[entry.Name()] {
				return fmt.Errorf(testUnregisteredFileDataFormat, entry.Name())
			}
			return err
		}); err != nil {
			t.Fatalf(testRoundErrorFormat, round, err)
		}
		daemon.Close()
	}
}

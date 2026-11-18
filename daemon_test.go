package yaagwadb

import (
	"bytes"
	"database/sql"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/errors"
)

const (
	// initial wait of the test retry
	testRetryBase = time.Millisecond
	// maximum wait of the test retry
	testRetryCap = 5 * time.Millisecond
	// maximum attempts of the test retry
	testRetryMaxAttempts = 3
	// time until the test retry gives up
	testRetryMaxDuration = time.Minute
	// maximum time eventually waits for the condition
	testEventuallyTimeout = 5 * time.Second
	// interval at which eventually rechecks the condition
	testEventuallyPollInterval = 5 * time.Millisecond

	// name of the file written in tests
	testFileName = "file"
	// name of the directory created in tests
	testDirectoryName = "directory"
	// content written in tests
	testFileData = "content"
	// permission of the file written in tests
	testFileMode = 0o644
	// permission of the directory created in tests
	testDirectoryMode = 0o755
	// owner UID / GID used in tests
	testOwnerID = 1000
	// UID / GID passed in tests that ignore ownership
	testNoOwnerID = 0
	// permission of temporary and index files created in tests
	testPrivateFileMode = 0o600
	// permission of directories created in tests (inside the data directory)
	testPrivateDirectoryMode = 0o700

	// name of the failure point called just before the index write commits
	testFailurePointBeforeCommit = "index-write:before-commit"
	// message of the error injected at the failure point
	testInjectedErrorMessage = "injected"
	// key prefix of contents in object storage
	testObjectFileDataPrefix = "filedata/"
	// name of the file data directory in the data directory
	testFileDataDirectoryName = "filedata"
	// name of the temporary file directory in the data directory
	testTemporaryDirectoryName = "temporary"
	// name of the index file in the data directory
	testDatabaseFileName = "yaagwa.db"
	// testDatabaseBusyTimeout is the lock wait of an SQLite file a test opens itself
	testDatabaseBusyTimeout = time.Second
	// number of tasks fetched per ListTasks call
	testListTasksLimit = 100
	// whether ListTasks returns only unfinished tasks
	testListActiveTasksOnly = true
	// whether ListTasks includes finished tasks
	testListAllTasks = false
	// expected count when exactly one exists
	testExpectedSingle = 1
	// detail attached to an error with fmt.Errorf
	testErrorDetailFormat = "%w: detail"
	// error message matching no constant
	testOtherErrorMessage = "other"
)

const (
	// repeated unit of the content used to check an in-progress index
	testRepeatedFileDataUnit = "yaagwa"
	// number of repetitions in the content
	testRepeatedFileDataCount = 10_000
	// ReadAt offset (start of the second repeated unit)
	testReadAtOffset = int64(len(testRepeatedFileDataUnit))
	// content before overwrite
	testFirstFileData = "first"
	// content after overwrite
	testSecondFileData = "second version"
	// name and content of the file created in the directory
	testInnerFileName = "a"
	// name of the file after the move
	testRenamedFileName = "b"
	// name used when moving a directory into itself
	testInsideName = "inside"
	// name and content of the rename source file
	testRenameSourceName     = "source"
	testRenameSourceFileData = "new"
	// name and content of the rename target (replaced) file
	testRenameTargetName     = "target"
	testRenameTargetFileData = "old"
	// permission set by Setattr
	testChangedFileMode = 0o600
	// UID set by Setattr
	testChangedUID = 42
	// extended attribute name and value
	testExtendedAttributeName  = "user.case"
	testExtendedAttributeValue = "A-1"
	// retry wait for failed PUTs while object storage is unreachable (longer than the test runs)
	testLongRetryWait = time.Hour
	// files written while PUTs to object storage are pending
	testPendingFirstName      = "first"
	testPendingFirstFileData  = "1"
	testPendingSecondName     = "second"
	testPendingSecondFileData = "2"
	// character and length for building a too-long name
	testLongNameCharacter = "x"
	testLongNameLength    = 256
	// invalid name examples (current, parent, with separator, with NUL)
	testCurrentDirectoryName = "."
	testParentDirectoryName  = ".."
	testNameWithSeparator    = "a/b"
	testNameWithNull         = "a\x00b"
	// valid name example
	testValidName = "ok.txt"
	// text appended after the queue name in waitIdle error messages
	testActiveTasksMessage = " has active tasks"
)

const (
	// failure message format when NotFound is expected
	testWantNotFoundFormat = "want NotFound, got %v"
	// failure message format when InvalidArgument is expected
	testWantInvalidArgumentFormat = "want InvalidArgument, got %v"
	// testWantWriteFailureMessage reports that Write succeeded although index registration failed
	testWantWriteFailureMessage = "want Write to fail when registration fails"
	// testFileDataCountFormat is the format used when file data remain
	testFileDataCountFormat = "%d file data remain"
	// failure message format when Corrupted is expected
	testWantCorruptedFormat = "want Corrupted, got %v"
	// failure message format when read content is unexpected
	testGotQuotedFormat = "got %q"
	// failure message format when the directory listing is unexpected
	testEntriesFormat = "entries %+v %v"
	// failure message format when the extended attribute name list is unexpected
	testExtendedAttributeNamesFormat = "names %v %v"
	// failure message format when the status is unexpected
	testStatusFormat = "status %+v %v"
	// failure message format when the task list is unexpected
	testTasksFormat = "tasks %+v %v"
	// Failure message format when object storage holds objects it should not
	testObjectsPutFormat = "objects were put: %v"
	// failure message format when file data remain
	testFileDataRemainsFormat = "file data remains: %v"
	// empty string (empty name, empty error text, or unset)
	testEmptyString = ""
	// zero count / zero bytes
	testZeroCount = 0
	// index of the first slice element
	testFirstElementIndex = 0
)

const (
	// format when the written size differs
	testWrittenSizeFormat = "size %d"
	// format when the inode is unexpected
	testInodeFormat = "inode %+v"
	// message when read content differs from the written content
	testContentMismatchMessage = "content mismatch"
	// format when the ReadAt result differs
	testReadAtFormat = "ReadAt %q %v"
	// format when inodes before and after overwrite are unexpected
	testFirstAndSecondInodeFormat = "first %+v second %+v"
	// format when the pre-overwrite file data is not logically deleted
	testOldFileDataFormat = "old file data %+v"
	// format when the second Unlink does not return NotFound
	testSecondUnlinkFormat = "second unlink: %v"
	// format when the directory mode differs
	testDirectoryModeFormat = "mode %o"
	// format when Exists is expected
	testWantExistsFormat = "want Exists, got %v"
	// format when IsDirectory is expected
	testWantIsDirectoryFormat = "want IsDirectory, got %v"
	// format when NotEmpty is expected
	testWantNotEmptyFormat = "want NotEmpty, got %v"
	// format when Locked is expected
	testWantLockedFormat = "want Locked, got %v"
	// format when a directory was moved into itself
	testMoveIntoItselfFormat = "move into itself: %v"
	// format when the inode and error are unexpected
	testInodeAndErrorFormat = "inode %+v %v"
	// format when the replaced file data is not logically deleted
	testReplacedFileDataFormat = "replaced file data %+v"
	// format when changed attributes differ
	testAttributesAfterFormat = "after %+v %v"
	// format when the extended attribute value differs
	testExtendedAttributeValueFormat = "value %q %v"
	// format when attributes after truncation differ
	testTruncatedFormat = "truncated %+v %v"
	// format when ValidateName accepts an invalid name
	testValidateNameFormat = "ValidateName(%q) = %v"
)

func testConfig(directory string) Config {
	fast := RetryPolicy{Base: testRetryBase, Cap: testRetryCap, MaxAttempts: testRetryMaxAttempts, MaxDuration: testRetryMaxDuration}
	return Config{
		DataDirectory:           directory,
		DisableScheduler:        true,
		FileDataDeleteRetry:     fast,
		ObjectStorageWriteRetry: fast,
		Logger:                  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func openTestDaemon(t *testing.T, config Config) *Daemon {
	t.Helper()
	daemon, err := Open(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { daemon.Close() })
	return daemon
}

func waitIdle(t *testing.T, daemon *Daemon) {
	t.Helper()
	eventually(t, func() error {
		return daemon.index.view(func(transaction *sql.Tx) error {
			for _, name := range queueNames {
				for _, state := range []TaskState{TaskPending, TaskRunning} {
					count, err := countTasks(transaction, name, state)
					if err != nil {
						return err
					}
					if count != testZeroCount {
						return errors.New(name + testActiveTasksMessage)
					}
				}
			}
			return nil
		})
	})
}

func eventually(t *testing.T, check func() error) {
	t.Helper()
	deadline := time.Now().Add(testEventuallyTimeout)
	for {
		err := check()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(testEventuallyPollInterval)
	}
}

func writeAndWait(t *testing.T, daemon *Daemon, parent uint64, name string, content []byte) *Inode {
	t.Helper()
	inode, err := daemon.Write(parent, name, testFileMode, testOwnerID, testOwnerID, bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	return inode
}

func readAll(t *testing.T, daemon *Daemon, inodeNumber uint64) []byte {
	t.Helper()
	file, err := daemon.OpenFileData(inodeNumber)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestWriteIsVisibleWhenResponded(t *testing.T) {
	daemon := openTestDaemon(t, testConfig(t.TempDir()))

	content := bytes.Repeat([]byte(testRepeatedFileDataUnit), testRepeatedFileDataCount)
	written, err := daemon.Write(RootInodeNumber, testFileName, testFileMode, testNoOwnerID, testNoOwnerID, bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if written.Size != uint64(len(content)) {
		t.Fatalf(testWrittenSizeFormat, written.Size)
	}
	// already in the index when the call returns
	inode, err := daemon.Lookup(RootInodeNumber, testFileName)
	if err != nil {
		t.Fatal(err)
	}
	if inode.InodeNumber != written.InodeNumber || inode.Size != uint64(len(content)) || inode.FileDataID != written.FileDataID {
		t.Fatalf(testInodeFormat, inode)
	}
	if got := readAll(t, daemon, inode.InodeNumber); !bytes.Equal(got, content) {
		t.Fatal(testContentMismatchMessage)
	}
	part := make([]byte, len(testRepeatedFileDataUnit))
	if _, err := daemon.ReadAt(inode.InodeNumber, part, testReadAtOffset); err != nil || string(part) != testRepeatedFileDataUnit {
		t.Fatalf(testReadAtFormat, part, err)
	}
}

func TestFailedRegistrationRemovesFileData(t *testing.T) {
	config := testConfig(t.TempDir())
	config.FailurePoint = func(name string) error {
		if name == testFailurePointBeforeCommit {
			return errors.New(testInjectedErrorMessage)
		}
		return nil
	}
	daemon := openTestDaemon(t, config)
	if _, err := daemon.Write(RootInodeNumber, testFileName, testFileMode, testNoOwnerID, testNoOwnerID, bytes.NewReader([]byte(testFileData))); err == nil {
		t.Fatal(testWantWriteFailureMessage)
	}
	// nothing remains in the index or task queues, and the file data is gone
	if _, err := daemon.Lookup(RootInodeNumber, testFileName); !Has(err, NotFound) {
		t.Fatalf(testWantNotFoundFormat, err)
	}
	if count := countFileDataFiles(t, daemon); count != testZeroCount {
		t.Fatalf(testFileDataCountFormat, count)
	}
}

func TestFailedRegistrationLeavesNoPutTask(t *testing.T) {
	storage := newMemoryObjectStorage(t)
	config := testConfig(t.TempDir())
	config.ObjectStorage = storage
	config.FailurePoint = func(name string) error {
		if name == testFailurePointBeforeCommit {
			return errors.New(testInjectedErrorMessage)
		}
		return nil
	}
	daemon := openTestDaemon(t, config)
	if _, err := daemon.Write(RootInodeNumber, testFileName, testFileMode, testNoOwnerID, testNoOwnerID, bytes.NewReader([]byte(testFileData))); err == nil {
		t.Fatal(testWantWriteFailureMessage)
	}
	// The put task is enqueued in the registration transaction, so it is rolled back with it
	waitIdle(t, daemon)
	tasks, err := daemon.ListTasks(ObjectStorageWriteQueue, testListAllTasks, testListTasksLimit)
	if err != nil || len(tasks) != testZeroCount {
		t.Fatalf(testTasksFormat, tasks, err)
	}
	storage.mutex.Lock()
	defer storage.mutex.Unlock()
	if len(storage.objects) != testZeroCount {
		t.Fatalf(testObjectsPutFormat, storage.objects)
	}
}

func countFileDataFiles(t *testing.T, daemon *Daemon) int {
	t.Helper()
	count := 0
	if err := filepath.WalkDir(daemon.fileDataDirectory, func(_ string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			count++
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestOverwriteKeepsInodeAndLogicallyDeletesOldFileData(t *testing.T) {
	daemon := openTestDaemon(t, testConfig(t.TempDir()))
	first := writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testFirstFileData))
	second := writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testSecondFileData))
	if second.InodeNumber != first.InodeNumber || second.FileDataID == first.FileDataID {
		t.Fatalf(testFirstAndSecondInodeFormat, first, second)
	}
	if got := readAll(t, daemon, second.InodeNumber); string(got) != testSecondFileData {
		t.Fatalf(testGotQuotedFormat, got)
	}
	if err := daemon.index.view(func(transaction *sql.Tx) error {
		fileData, err := getFileData(transaction, first.FileDataID)
		if err != nil {
			return err
		}
		if fileData.State != FileDataLogicallyDeleted || fileData.LogicallyDeletedAt == testZeroCount {
			t.Errorf(testOldFileDataFormat, fileData)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestUnlinkIsLogicalDelete(t *testing.T) {
	daemon := openTestDaemon(t, testConfig(t.TempDir()))
	inode := writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testFileData))
	if err := daemon.Unlink(RootInodeNumber, testFileName); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	if _, err := daemon.Lookup(RootInodeNumber, testFileName); !Has(err, NotFound) {
		t.Fatalf(testWantNotFoundFormat, err)
	}
	// physical deletion happens after the grace period, so the file data remains
	if _, err := os.Stat(FileDataPath(daemon.fileDataDirectory, inode.FileDataID)); err != nil {
		t.Fatal(err)
	}
	if err := daemon.Unlink(RootInodeNumber, testFileName); !Has(err, NotFound) {
		t.Fatalf(testSecondUnlinkFormat, err)
	}
}

func TestDirectories(t *testing.T) {
	daemon := openTestDaemon(t, testConfig(t.TempDir()))
	directory, err := daemon.Mkdir(RootInodeNumber, testDirectoryName, testDirectoryMode, testNoOwnerID, testNoOwnerID)
	if err != nil {
		t.Fatal(err)
	}
	if !directory.IsDirectory() {
		t.Fatalf(testDirectoryModeFormat, directory.Mode)
	}
	if _, err := daemon.Mkdir(RootInodeNumber, testDirectoryName, testDirectoryMode, testNoOwnerID, testNoOwnerID); !Has(err, Exists) {
		t.Fatalf(testWantExistsFormat, err)
	}
	writeAndWait(t, daemon, directory.InodeNumber, testInnerFileName, []byte(testInnerFileName))
	if _, err := daemon.Write(RootInodeNumber, testDirectoryName, testFileMode, testNoOwnerID, testNoOwnerID, bytes.NewReader(nil)); !Has(err, IsDirectory) {
		t.Fatalf(testWantIsDirectoryFormat, err)
	}
	if err := daemon.Rmdir(RootInodeNumber, testDirectoryName); !Has(err, NotEmpty) {
		t.Fatalf(testWantNotEmptyFormat, err)
	}
	entries, err := daemon.ListDirectory(directory.InodeNumber)
	if err != nil || len(entries) != testExpectedSingle || entries[testFirstElementIndex].Name != testInnerFileName {
		t.Fatalf(testEntriesFormat, entries, err)
	}

	if err := daemon.Rename(directory.InodeNumber, testInnerFileName, RootInodeNumber, testRenamedFileName); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	if _, err := daemon.Lookup(RootInodeNumber, testRenamedFileName); err != nil {
		t.Fatal(err)
	}
	if err := daemon.Rename(RootInodeNumber, testDirectoryName, directory.InodeNumber, testInsideName); !Has(err, InvalidArgument) {
		t.Fatalf(testMoveIntoItselfFormat, err)
	}
	if err := daemon.Rmdir(RootInodeNumber, testDirectoryName); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	if _, err := daemon.Lookup(RootInodeNumber, testDirectoryName); !Has(err, NotFound) {
		t.Fatalf(testWantNotFoundFormat, err)
	}
}

func TestRenameReplacesTarget(t *testing.T) {
	daemon := openTestDaemon(t, testConfig(t.TempDir()))
	source := writeAndWait(t, daemon, RootInodeNumber, testRenameSourceName, []byte(testRenameSourceFileData))
	target := writeAndWait(t, daemon, RootInodeNumber, testRenameTargetName, []byte(testRenameTargetFileData))
	if err := daemon.Rename(RootInodeNumber, testRenameSourceName, RootInodeNumber, testRenameTargetName); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	inode, err := daemon.Lookup(RootInodeNumber, testRenameTargetName)
	if err != nil || inode.InodeNumber != source.InodeNumber {
		t.Fatalf(testInodeAndErrorFormat, inode, err)
	}
	if err := daemon.index.view(func(transaction *sql.Tx) error {
		fileData, err := getFileData(transaction, target.FileDataID)
		if err == nil && fileData.State != FileDataLogicallyDeleted {
			t.Errorf(testReplacedFileDataFormat, fileData)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

func TestAttributesAndExtendedAttributes(t *testing.T) {
	daemon := openTestDaemon(t, testConfig(t.TempDir()))
	inode := writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testFileData))

	changed, err := daemon.Setattr(inode.InodeNumber, AttributeMode|AttributeUID, Attribute{Mode: testChangedFileMode, UID: testChangedUID})
	if err != nil {
		t.Fatal(err)
	}
	// returns attributes already applied at response time
	if changed.Mode != ModeRegular|testChangedFileMode || changed.UID != testChangedUID {
		t.Fatalf(testAttributesAfterFormat, changed, err)
	}
	if err := daemon.SetExtendedAttribute(inode.InodeNumber, testExtendedAttributeName, []byte(testExtendedAttributeValue)); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	after, err := daemon.GetAttribute(inode.InodeNumber)
	if err != nil || after.Mode != ModeRegular|testChangedFileMode || after.UID != testChangedUID {
		t.Fatalf(testAttributesAfterFormat, after, err)
	}
	value, err := daemon.GetExtendedAttribute(inode.InodeNumber, testExtendedAttributeName)
	if err != nil || string(value) != testExtendedAttributeValue {
		t.Fatalf(testExtendedAttributeValueFormat, value, err)
	}
	names, err := daemon.ListExtendedAttributes(inode.InodeNumber)
	if err != nil || len(names) != testExpectedSingle {
		t.Fatalf(testExtendedAttributeNamesFormat, names, err)
	}
	if err := daemon.RemoveExtendedAttribute(inode.InodeNumber, testExtendedAttributeName); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	if _, err := daemon.GetExtendedAttribute(inode.InodeNumber, testExtendedAttributeName); !Has(err, NotFound) {
		t.Fatalf(testWantNotFoundFormat, err)
	}

	if err := daemon.Truncate(inode.InodeNumber); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	truncated, err := daemon.GetAttribute(inode.InodeNumber)
	if err != nil || truncated.Size != testZeroCount || truncated.Mode != ModeRegular|testChangedFileMode {
		t.Fatalf(testTruncatedFormat, truncated, err)
	}
	if got := readAll(t, daemon, inode.InodeNumber); len(got) != testZeroCount {
		t.Fatalf(testGotQuotedFormat, got)
	}
}

// Writes are accepted while PUTs to object storage are pending. The backlog is watched through Status, not refused.
func TestWriteWhileObjectStoragePutsPending(t *testing.T) {
	config := testConfig(t.TempDir())
	// while PUTs keep failing, tasks stay queued awaiting retry
	config.ObjectStorageWriteRetry = RetryPolicy{Base: testLongRetryWait, Cap: testLongRetryWait,
		MaxAttempts: testRetryMaxAttempts, MaxDuration: testLongRetryWait}
	daemon := openWithFailingStorage(t, config, &failingObjectStorage{})
	if _, err := daemon.Write(RootInodeNumber, testPendingFirstName, testFileMode, testNoOwnerID, testNoOwnerID, bytes.NewReader([]byte(testPendingFirstFileData))); err != nil {
		t.Fatal(err)
	}
	inode, err := daemon.Write(RootInodeNumber, testPendingSecondName, testFileMode, testNoOwnerID, testNoOwnerID, bytes.NewReader([]byte(testPendingSecondFileData)))
	if err != nil {
		t.Fatal(err)
	}
	if got := readAll(t, daemon, inode.InodeNumber); string(got) != testPendingSecondFileData {
		t.Fatalf(testGotQuotedFormat, got)
	}
}

func TestDataDirectoryLock(t *testing.T) {
	directory := t.TempDir()
	openTestDaemon(t, testConfig(directory))
	if _, err := Open(testConfig(directory)); !Has(err, Locked) {
		t.Fatalf(testWantLockedFormat, err)
	}
}

func TestValidateName(t *testing.T) {
	for _, name := range []string{testEmptyString, testCurrentDirectoryName, testParentDirectoryName, testNameWithSeparator, testNameWithNull, string(bytes.Repeat([]byte(testLongNameCharacter), testLongNameLength))} {
		if err := ValidateName(name); !Has(err, InvalidArgument) {
			t.Errorf(testValidateNameFormat, name, err)
		}
	}
	if err := ValidateName(testValidName); err != nil {
		t.Fatal(err)
	}
}

func TestRefuseLegacyIndex(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, legacyIndexFileName), nil, testPrivateFileMode); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(testConfig(directory)); !Has(err, NotSupported) {
		t.Fatalf(testWantNotSupportedFormat, err)
	}
}

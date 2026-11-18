package yaagwadb

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

const (
	// File data read after the restore
	testRestoredFileData = "restored content"
	// File included in the index archive (name equals content)
	testKeptName = "kept"
	// File written after the archive and therefore lost by the restore (name equals content)
	testLostName = "lost"
	// File name pattern of the index moved aside by a forced restore
	testBrokenIndexPattern = testDatabaseFileName + ".broken-*"
	// Number of indexes moved aside
	testExpectedBrokenIndexCount = 1
	// Content of a file data nothing references
	testOrphanFileData = "orphan"
	// Name and content of a temporary file left by a crash
	testLeftoverTemporaryFileName = "write-123"
	testLeftoverTemporaryFileData = "half"
	// Name of the failure point hit just before enqueuing tasks
	testFailurePointBeforeEnqueue = "write:before-enqueue"
	// Pattern for finding file data (files under two directory levels)
	testFileDataGlob = "*"

	// Failure message format when a file data already exists instead of being fetched lazily
	testLazyDownloadFormat = "file data should be downloaded lazily: %v"
	// Failure message format when the number of indexes moved aside differs
	testBrokenIndexFormat = "broken index %v"
	// Failure message format when a file the startup cleanup should remove remains
	testShouldBeRemovedFormat = "%s should be removed: %v"
	// Failure message when an error was expected but the call succeeded
	testWantErrorMessage = "want error"
	// Failure message format when file data remain (plural)
	testFileDataRemainFormat = "file data remain: %v"
)

func TestRestoreFromObjectStorage(t *testing.T) {
	storage := newMemoryObjectStorage(t)
	config := testConfig(t.TempDir())
	config.ObjectStorage = storage
	daemon, err := Open(config)
	if err != nil {
		t.Fatal(err)
	}
	inode := writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testRestoredFileData))
	if err := daemon.putIndexArchive(); err != nil {
		t.Fatal(err)
	}
	daemon.Close()

	fresh := testConfig(t.TempDir())
	fresh.ObjectStorage = storage
	restored := openTestDaemon(t, fresh)
	found, err := restored.Lookup(RootInodeNumber, testFileName)
	if err != nil {
		t.Fatal(err)
	}
	path := FileDataPath(restored.fileDataDirectory, inode.FileDataID)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf(testLazyDownloadFormat, err)
	}
	if got := readAll(t, restored, found.InodeNumber); string(got) != testRestoredFileData {
		t.Fatalf(testGotQuotedFormat, got)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreForceMovesLocalDataAside(t *testing.T) {
	storage := newMemoryObjectStorage(t)
	directory := t.TempDir()
	config := testConfig(directory)
	config.ObjectStorage = storage
	daemon, err := Open(config)
	if err != nil {
		t.Fatal(err)
	}
	writeAndWait(t, daemon, RootInodeNumber, testKeptName, []byte(testKeptName))
	if err := daemon.putIndexArchive(); err != nil {
		t.Fatal(err)
	}
	writeAndWait(t, daemon, RootInodeNumber, testLostName, []byte(testLostName))
	daemon.Close()

	restored, err := RestoreForce(config)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if _, err := restored.Lookup(RootInodeNumber, testKeptName); err != nil {
		t.Fatal(err)
	}
	if _, err := restored.Lookup(RootInodeNumber, testLostName); !Has(err, NotFound) {
		t.Fatalf(testWantNotFoundFormat, err)
	}
	broken, _ := filepath.Glob(filepath.Join(directory, testBrokenIndexPattern))
	if len(broken) != testExpectedBrokenIndexCount {
		t.Fatalf(testBrokenIndexFormat, broken)
	}
}

func TestStartupCleanup(t *testing.T) {
	directory := t.TempDir()
	config := testConfig(directory)
	daemon, err := Open(config)
	if err != nil {
		t.Fatal(err)
	}
	inode := writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testFileData))
	daemon.Close()

	// Simulate a file data and a temporary file left by a crash after rename and before enqueuing tasks
	orphan := newFileDataID()
	orphanPath := FileDataPath(filepath.Join(directory, testFileDataDirectoryName), orphan)
	if err := os.MkdirAll(filepath.Dir(orphanPath), testPrivateDirectoryMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(orphanPath, []byte(testOrphanFileData), testPrivateFileMode); err != nil {
		t.Fatal(err)
	}
	temporary := filepath.Join(directory, testTemporaryDirectoryName, testLeftoverTemporaryFileName)
	if err := os.WriteFile(temporary, []byte(testLeftoverTemporaryFileData), testPrivateFileMode); err != nil {
		t.Fatal(err)
	}

	reopened := openTestDaemon(t, config)
	for _, path := range []string{orphanPath, temporary} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf(testShouldBeRemovedFormat, path, err)
		}
	}
	if got := readAll(t, reopened, inode.InodeNumber); !bytes.Equal(got, []byte(testFileData)) {
		t.Fatalf(testGotQuotedFormat, got)
	}
}

func TestCrashBeforeEnqueue(t *testing.T) {
	directory := t.TempDir()
	config := testConfig(directory)
	config.FailurePoint = func(name string) error {
		if name == testFailurePointBeforeEnqueue {
			return os.ErrClosed
		}
		return nil
	}
	daemon, err := Open(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := daemon.Write(RootInodeNumber, testFileName, testFileMode, testNoOwnerID, testNoOwnerID, bytes.NewReader([]byte(testFileData))); err == nil {
		t.Fatal(testWantErrorMessage)
	}
	daemon.Close()
	reopened := openTestDaemon(t, testConfig(directory))
	if _, err := reopened.Lookup(RootInodeNumber, testFileName); !Has(err, NotFound) {
		t.Fatalf(testWantNotFoundFormat, err)
	}
	entries, _ := filepath.Glob(filepath.Join(reopened.fileDataDirectory, testFileDataGlob, testFileDataGlob, testFileDataGlob))
	if len(entries) != testZeroCount {
		t.Fatalf(testFileDataRemainFormat, entries)
	}
}

package yaagwadb

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

const (
	// base directory for building file data paths
	testFileDataRoot = "root"
	// length of the first- and second-level directory names taken from the start of the fileDataID
	testFileDataDirectoryNameLength = 2
	// testFileDataSecondDirectoryEnd is where the second-level directory name ends in the file data ID.
	testFileDataSecondDirectoryEnd = 2 * testFileDataDirectoryNameLength
	// invalid fileDataID example (too short)
	testShortFileDataID = "xyz"
	// non-hexadecimal character
	testNonHexCharacter = "g"
	// opaque content example
	testOpaqueFileData = "opaque bytes"
	// one-byte content
	testSingleByteFileData = "x"
	// number of repeated deletions (idempotency check)
	testRepeatedDeleteCount = 2
	// invalid fileDataID pointing outside the directory
	testEscapingFileDataID = "../x"

	// failure message format when a generated fileDataID is invalid
	testFileDataIDFormat = "fileDataID %q"
	// failure message format when the file data path differs
	testFileDataPathFormat = "path %s, want %s"
	// failure message format when an invalid fileDataID is accepted
	testValidFileDataIDFormat = "validFileDataID(%q)"
	// failure message format when size and hash differ
	testSizeAndHashFormat = "size %d hash %x"
	// failure message format when stored content differs
	testStoredFormat = "stored %q %v"
	// failure message format when temporary files remain
	testTemporaryFilesRemainFormat = "temporary files remain: %v"
)

func TestFileDataIDAndPath(t *testing.T) {
	fileDataID := newFileDataID()
	if !validFileDataID(fileDataID) || newFileDataID() == fileDataID {
		t.Fatalf(testFileDataIDFormat, fileDataID)
	}
	want := filepath.Join(testFileDataRoot, fileDataID[:testFileDataDirectoryNameLength], fileDataID[testFileDataDirectoryNameLength:testFileDataSecondDirectoryEnd], fileDataID)
	if got := FileDataPath(testFileDataRoot, fileDataID); got != want {
		t.Fatalf(testFileDataPathFormat, got, want)
	}
	for _, invalid := range []string{testEmptyString, testShortFileDataID, fileDataID[:fileDataIDLength-len(testNonHexCharacter)] + testNonHexCharacter} {
		if validFileDataID(invalid) {
			t.Errorf(testValidFileDataIDFormat, invalid)
		}
	}
}

func TestWriteFileData(t *testing.T) {
	daemon := openTestDaemon(t, testConfig(t.TempDir()))
	content := []byte(testOpaqueFileData)
	fileDataID, size, hash, err := daemon.writeFileData(bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if size != uint64(len(content)) || hash != sha256.Sum256(content) {
		t.Fatalf(testSizeAndHashFormat, size, hash)
	}
	stored, err := os.ReadFile(FileDataPath(daemon.fileDataDirectory, fileDataID))
	if err != nil || !bytes.Equal(stored, content) {
		t.Fatalf(testStoredFormat, stored, err)
	}
	// no temporary files remain
	if entries, _ := os.ReadDir(daemon.temporaryDirectory); len(entries) != testZeroCount {
		t.Fatalf(testTemporaryFilesRemainFormat, entries)
	}
}

func TestFileDataDeleteIsIdempotent(t *testing.T) {
	daemon := openTestDaemon(t, testConfig(t.TempDir()))
	fileDataID, _, _, err := daemon.writeFileData(bytes.NewReader([]byte(testSingleByteFileData)))
	if err != nil {
		t.Fatal(err)
	}
	for range testRepeatedDeleteCount {
		if err := daemon.unlinkFileData(fileDataID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(FileDataPath(daemon.fileDataDirectory, fileDataID)); !os.IsNotExist(err) {
		t.Fatalf(testFileDataRemainsFormat, err)
	}
	if err := daemon.unlinkFileData(testEscapingFileDataID); !Has(err, InvalidArgument) {
		t.Fatalf(testWantInvalidArgumentFormat, err)
	}
}

func TestOpenFileDataWithoutObjectStorage(t *testing.T) {
	daemon := openTestDaemon(t, testConfig(t.TempDir()))
	if _, err := daemon.openFileData(newFileDataID()); !Has(err, Corrupted) {
		t.Fatalf(testWantCorruptedFormat, err)
	}
}

package yaagwadb

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	// fileDataIDRandomByteLength is the number of random bytes a fileDataID is built from.
	fileDataIDRandomByteLength = 16
	// hexCharactersPerByte is the number of hex characters per byte.
	hexCharactersPerByte = 2
	// fileDataIDLength is the length of a fileDataID (hex-encoded random bytes).
	fileDataIDLength = fileDataIDRandomByteLength * hexCharactersPerByte
)

const (
	firstShardStart  = 0
	firstShardEnd    = 2 // first level is the first byte of the fileDataID
	secondShardStart = firstShardEnd
	secondShardEnd   = 4 // second level is the next byte of the fileDataID
)

const (
	// writeTemporaryFilePattern is the name pattern of the temporary file Write writes the content to. os.CreateTemp fills in *.
	writeTemporaryFilePattern = "write-*"
	// failurePointWriteAfterSync is a crash injection failure point: after fsync of the temporary file, before rename.
	failurePointWriteAfterSync = "write:after-sync"
)

const (
	fileDataErrorFormatMissingLocally         = "%w: file data %s is missing locally"
	fileDataErrorFormatMissingInObjectStorage = "%w: file data %s is missing in object storage"
	fileDataErrorFormatInvalidFileDataID      = "%w: fileDataID %q"
)

func newFileDataID() string {
	buffer := make([]byte, fileDataIDRandomByteLength)
	_, _ = rand.Read(buffer) // crypto/rand.Read never fails
	return hex.EncodeToString(buffer)
}

func validFileDataID(fileDataID string) bool {
	if len(fileDataID) != fileDataIDLength {
		return false
	}
	_, err := hex.DecodeString(fileDataID)
	return err == nil
}

func FileDataPath(directory, fileDataID string) string {
	return filepath.Join(directory, fileDataID[firstShardStart:firstShardEnd], fileDataID[secondShardStart:secondShardEnd], fileDataID)
}

func (s *Daemon) writeFileData(content io.Reader) (fileDataID string, size uint64, hash [fileDataHashLength]byte, err error) {
	temporary, err := os.CreateTemp(s.temporaryDirectory, writeTemporaryFilePattern)
	if err != nil {
		return "", 0, hash, err
	}
	defer os.Remove(temporary.Name()) // no-op after rename
	digest := sha256.New()
	written, err := io.Copy(io.MultiWriter(temporary, digest), content)
	if err == nil {
		err = temporary.Sync()
	}
	if closeError := temporary.Close(); err == nil {
		err = closeError
	}
	if err != nil {
		return "", 0, hash, err
	}
	if err := s.failurePoint(failurePointWriteAfterSync); err != nil {
		return "", 0, hash, err
	}
	fileDataID = newFileDataID()
	if err := s.placeFileData(temporary.Name(), fileDataID); err != nil {
		return "", 0, hash, err
	}
	copy(hash[:], digest.Sum(nil))
	return fileDataID, uint64(written), hash, nil
}

func (s *Daemon) placeFileData(source, fileDataID string) error {
	path := FileDataPath(s.fileDataDirectory, fileDataID)
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, dataDirectoryPermission); err != nil {
		return err
	}
	if err := os.Rename(source, path); err != nil {
		return err
	}
	return syncDirectory(directory)
}

func (s *Daemon) openFileData(fileDataID string) (*os.File, error) {
	file, err := os.Open(FileDataPath(s.fileDataDirectory, fileDataID))
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return file, err
	}
	if s.objectStorage == nil {
		return nil, fmt.Errorf(fileDataErrorFormatMissingLocally, errors.New(Corrupted), fileDataID)
	}
	if err := s.downloadFileData(fileDataID); err != nil {
		if Has(err, NotFound) {
			return nil, fmt.Errorf(fileDataErrorFormatMissingInObjectStorage, errors.New(Corrupted), fileDataID)
		}
		return nil, err
	}
	return os.Open(FileDataPath(s.fileDataDirectory, fileDataID))
}

func (s *Daemon) unlinkFileData(fileDataID string) error {
	if !validFileDataID(fileDataID) {
		return fmt.Errorf(fileDataErrorFormatInvalidFileDataID, errors.New(InvalidArgument), fileDataID)
	}
	if err := os.Remove(FileDataPath(s.fileDataDirectory, fileDataID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

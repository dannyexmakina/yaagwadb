package yaagwadb

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	objectstorage "yaagwadb/object_storage"
)

const (
	objectFileDataPrefix = "filedata/"
	objectIndexLatest    = "index/latest"
	// objectKeySeparator separates ObjectStoragePrefix from the key.
	objectKeySeparator = "/"
	// objectStorageWithoutPrefix means ObjectStoragePrefix is not set.
	objectStorageWithoutPrefix = ""
)

const (
	indexArchiveTemporaryPattern = "index-archive-*"
	downloadTemporaryPattern     = "download-*"
)

const (
	transferWrappedErrorFormat      = "%w: %v"
	transferInvalidFileDataIDFormat = "%w: fileDataID %q"
	transferSizeMismatchFormat      = "%w: uploaded file data %s size mismatch: %d != %d"
)

func (s *Daemon) objectKey(path string) string {
	if s.config.ObjectStoragePrefix == objectStorageWithoutPrefix {
		return path
	}
	return strings.TrimSuffix(s.config.ObjectStoragePrefix, objectKeySeparator) + objectKeySeparator + path
}

func (s *Daemon) fileDataObjectKey(fileDataID string) string {
	return s.objectKey(objectFileDataPrefix + fileDataID)
}

func fromObjectStorage(err error) error {
	if objectstorage.Has(err, objectstorage.NotFound) {
		return fmt.Errorf(transferWrappedErrorFormat, errors.New(NotFound), err)
	}
	if objectstorage.Has(err, objectstorage.InvalidArgument) {
		return fmt.Errorf(transferWrappedErrorFormat, errors.New(InvalidArgument), err)
	}
	return err
}

func (s *Daemon) uploadFileData(fileDataID string) error {
	if !validFileDataID(fileDataID) {
		return fmt.Errorf(transferInvalidFileDataIDFormat, errors.New(InvalidArgument), fileDataID)
	}
	key := s.fileDataObjectKey(fileDataID)
	file, err := os.Open(FileDataPath(s.fileDataDirectory, fileDataID))
	if errors.Is(err, os.ErrNotExist) {
		if _, err := s.objectStorage.Head(s.context, key); err != nil {
			return fromObjectStorage(err)
		}
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if err := s.objectStorage.Put(s.context, key, file, info.Size()); err != nil {
		return fromObjectStorage(err)
	}
	remoteSize, err := s.objectStorage.Head(s.context, key)
	if err != nil {
		return fromObjectStorage(err)
	}
	if remoteSize != info.Size() {
		return fmt.Errorf(transferSizeMismatchFormat, errors.New(Corrupted), fileDataID, remoteSize, info.Size())
	}
	return nil
}

func (s *Daemon) deleteFileDataInObjectStorage(fileDataID string) error {
	if !validFileDataID(fileDataID) {
		return fmt.Errorf(transferInvalidFileDataIDFormat, errors.New(InvalidArgument), fileDataID)
	}
	if err := s.objectStorage.Delete(s.context, s.fileDataObjectKey(fileDataID)); err != nil &&
		!objectstorage.Has(err, objectstorage.NotFound) {
		return fromObjectStorage(err)
	}
	return nil
}

func (s *Daemon) putIndexArchive() error {
	directory, err := os.MkdirTemp(s.temporaryDirectory, indexArchiveTemporaryPattern)
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, s.config.DatabaseFileName)
	// VACUUM INTO writes a consistent copy of the index without stopping writes
	if _, err := s.index.read.Exec(`VACUUM INTO ?`, path); err != nil {
		return err
	}
	if err := clearTaskQueues(path, s.config.DatabaseBusyTimeout); err != nil {
		return err
	}
	archive, err := os.Open(path)
	if err != nil {
		return err
	}
	defer archive.Close()
	information, err := archive.Stat()
	if err != nil {
		return err
	}
	return fromObjectStorage(s.objectStorage.Put(s.context, s.objectKey(objectIndexLatest), archive, information.Size()))
}

func (s *Daemon) downloadFileData(fileDataID string) error {
	reader, err := s.objectStorage.Get(s.context, s.fileDataObjectKey(fileDataID))
	if err != nil {
		return fromObjectStorage(err)
	}
	defer reader.Close()
	temporary, err := os.CreateTemp(s.temporaryDirectory, downloadTemporaryPattern)
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	_, err = io.Copy(temporary, reader)
	if err == nil {
		err = temporary.Sync()
	}
	if closeError := temporary.Close(); err == nil {
		err = closeError
	}
	if err != nil {
		return err
	}
	return s.placeFileData(temporary.Name(), fileDataID)
}

func clearTaskQueues(path string, busyTimeout time.Duration) error {
	// The backup carries the index only: a restore starts with empty task queues
	archive, err := openDatabase(path, busyTimeout)
	if err != nil {
		return err
	}
	err = archive.update(func(transaction *sql.Tx) error {
		for _, name := range queueNames {
			if _, err := transaction.Exec(`DELETE FROM ` + name); err != nil {
				return err
			}
		}
		return nil
	})
	if err == nil {
		_, err = archive.write.Exec(`VACUUM`)
	}
	// Closing the last connection merges the WAL of the copy into the file that is put
	if closeError := archive.close(); err == nil {
		err = closeError
	}
	return err
}

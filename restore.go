package yaagwadb

import (
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/pkg/errors"

	objectstorage "yaagwadb/object_storage"
)

const (
	// restoreTemporaryFilePattern is the name pattern of the temporary file for the downloaded index. os.CreateTemp fills in *.
	restoreTemporaryFilePattern = "restore-*"
	// brokenDataSuffixFormat is the suffix RestoreForce uses to keep broken local data under another name. %d is the time in nanoseconds.
	brokenDataSuffixFormat = ".broken-%d"
)

const (
	logMessageRestoredIndex               = "restored index from object storage"
	logMessageRemovedUnregisteredFileData = "removed unregistered file data"
	logMessageMovedLocalDataAside         = "moved local data aside"
	logKeyCount                           = "count"
	logKeyPath                            = "path"
)

const (
	restoreErrorFormatObjectStorageNotConfigured = "%w: object storage is not configured"
	restoreErrorFormatLockCause                  = "%w: %v" // attaches the flock failure cause
)

func (s *Daemon) restoreIndexIfMissing(path string) error {
	if _, err := os.Stat(path); err == nil || s.objectStorage == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	reader, err := s.objectStorage.Get(s.context, s.objectKey(objectIndexLatest))
	if objectstorage.Has(err, objectstorage.NotFound) {
		return nil // first start
	}
	if err != nil {
		return fromObjectStorage(err)
	}
	defer reader.Close()
	temporary, err := os.CreateTemp(s.temporaryDirectory, restoreTemporaryFilePattern)
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
	// Journal files left without their database would be applied to the restored one
	for _, suffix := range sqliteJournalSuffixes {
		if err := os.Remove(path + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.Rename(temporary.Name(), path); err != nil {
		return err
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	s.logger.Info(logMessageRestoredIndex)
	return nil
}

func (s *Daemon) removeTemporaryFiles() error {
	entries, err := os.ReadDir(s.temporaryDirectory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(s.temporaryDirectory, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (s *Daemon) removeUnregisteredFileData() error {
	var referenced map[string]bool
	if err := s.index.view(func(transaction *sql.Tx) error {
		var err error
		referenced, err = listFileDataIDs(transaction)
		return err
	}); err != nil {
		return err
	}
	// A file data an unfinished task refers to is left to that task
	if err := s.index.view(func(transaction *sql.Tx) error {
		for _, name := range queueNames {
			payloads, err := listActiveTaskPayloads(transaction, name)
			if err != nil {
				return err
			}
			for _, payload := range payloads {
				referenced[string(payload)] = true
			}
		}
		return nil
	}); err != nil {
		return err
	}
	removed := 0
	err := filepath.WalkDir(s.fileDataDirectory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if referenced[entry.Name()] {
			return nil
		}
		removed++
		return os.Remove(path)
	})
	if removed > 0 {
		s.logger.Info(logMessageRemovedUnregisteredFileData, logKeyCount, removed)
	}
	return err
}

func RestoreForce(config Config) (*Daemon, error) {
	config = config.withDefaults()
	if config.ObjectStorage == nil {
		return nil, fmt.Errorf(restoreErrorFormatObjectStorageNotConfigured, errors.New(InvalidArgument))
	}
	// check before moving local data aside, so that a configuration Open rejects does not leave the data moved
	if err := CheckConfig(config); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(config.DataDirectory, dataDirectoryPermission); err != nil {
		return nil, err
	}
	lockFile, err := lockDataDirectory(config.DataDirectory)
	if err != nil {
		return nil, err
	}
	suffix := fmt.Sprintf(brokenDataSuffixFormat, time.Now().UnixNano())
	for _, name := range localDataNames(config.DatabaseFileName) {
		path := filepath.Join(config.DataDirectory, name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if err := os.Rename(path, path+suffix); err != nil {
			lockFile.Close()
			return nil, err
		}
		config.Logger.Warn(logMessageMovedLocalDataAside, logKeyPath, path+suffix)
	}
	lockFile.Close()
	return Open(config)
}

func localDataNames(databaseFileName string) []string {
	names := []string{fileDataDirectoryName}
	names = append(names, databaseFileName)
	for _, suffix := range sqliteJournalSuffixes {
		names = append(names, databaseFileName+suffix)
	}
	return names
}

func lockDataDirectory(directory string) (*os.File, error) {
	lockFile, err := os.OpenFile(filepath.Join(directory, dataDirectoryLockFileName), os.O_RDWR|os.O_CREATE, dataDirectoryFilePermission)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lockFile.Close()
		return nil, fmt.Errorf(restoreErrorFormatLockCause, errors.New(Locked), err)
	}
	return lockFile, nil
}

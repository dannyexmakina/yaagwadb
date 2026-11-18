package yaagwadb

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

// taskKindUnknownFormat formats a task kind that has no display name.
const taskKindUnknownFormat = "TaskKind(%d)"

const (
	putFileDataKeyPrefix    = "put:"
	deleteFileDataKeyPrefix = "delete:"
)

// indexArchiveTaskKey is the key of the index archive task. Only one archive task is ever queued.
const indexArchiveTaskKey = "index-archive"

// failurePointIndexWriteBeforeCommit names the test failure point just before an index write commits.
const failurePointIndexWriteBeforeCommit = "index-write:before-commit"

const (
	fileDataDeleteUnsupportedKindFormat     = "%w: file data delete task kind %d"
	objectStorageWriteUnsupportedKindFormat = "%w: object storage write task kind %d"
)

const (
	TaskDeleteFileData TaskKind = 1
)

const (
	TaskPutFileData TaskKind = iota + 1
	TaskDeleteFileDataInObjectStorage
	TaskPutIndexArchive
)

const (
	taskKindUnusedName                        = ""
	taskKindDeleteFileDataName                = "DeleteFileData"
	taskKindPutFileDataName                   = "PutFileData"
	taskKindDeleteFileDataInObjectStorageName = "DeleteFileDataInObjectStorage"
	taskKindPutIndexArchiveName               = "PutIndexArchive"
)

var taskKindNames = map[string][]string{
	FileDataDeleteQueue:     {taskKindUnusedName, taskKindDeleteFileDataName},
	ObjectStorageWriteQueue: {taskKindUnusedName, taskKindPutFileDataName, taskKindDeleteFileDataInObjectStorageName, taskKindPutIndexArchiveName},
}

func TaskKindName(queue string, kind TaskKind) string {
	if names := taskKindNames[queue]; int(kind) < len(names) {
		return names[kind]
	}
	return fmt.Sprintf(taskKindUnknownFormat, kind)
}

// TaskRequests holds the tasks to enqueue per queue name. They are inserted in one transaction.
type TaskRequests map[string][]TaskRequest

func (r TaskRequests) add(queue string, requests ...TaskRequest) {
	r[queue] = append(r[queue], requests...)
}

func fileDataDeleteTask(fileDataID string) TaskRequest {
	return TaskRequest{Kind: TaskDeleteFileData, Key: fileDataID, Payload: []byte(fileDataID)}
}

func putFileDataTask(fileDataID string) TaskRequest {
	return TaskRequest{Kind: TaskPutFileData, Key: putFileDataKeyPrefix + fileDataID, Payload: []byte(fileDataID)}
}

func deleteFileDataInObjectStorageTask(fileDataID string) TaskRequest {
	return TaskRequest{Kind: TaskDeleteFileDataInObjectStorage, Key: deleteFileDataKeyPrefix + fileDataID, Payload: []byte(fileDataID)}
}

func indexArchiveTask() TaskRequest {
	return TaskRequest{Kind: TaskPutIndexArchive, Key: indexArchiveTaskKey}
}

type fileDataRegistration struct {
	Parent     uint64
	Name       string
	Mode       uint32
	UID, GID   uint32
	FileDataID string
	Size       uint64
	Hash       [sha256.Size]byte
	ModifyTime int64
}

type directoryCreation struct {
	Parent   uint64
	Name     string
	Mode     uint32
	UID, GID uint32
}

type renameTarget struct {
	OldParent uint64
	OldName   string
	NewParent uint64
	NewName   string
}

// AttributeMask selects the fields Setattr changes. Size changes replace the content, so they go through Write.
type AttributeMask uint32

const (
	AttributeMode AttributeMask = 1 << iota
	AttributeUID
	AttributeGID
	AttributeAccessTime
	AttributeModifyTime
)

// Attribute holds the Setattr values. Only fields in Mask are used.
type Attribute struct {
	Mode       uint32
	UID, GID   uint32
	AccessTime int64
	ModifyTime int64
}

type attributeChange struct {
	InodeNumber uint64
	Mask        AttributeMask
	Attribute   Attribute
}

func (s *Daemon) executeFileDataDelete(task *Task) error {
	if task.Kind != TaskDeleteFileData {
		return fmt.Errorf(fileDataDeleteUnsupportedKindFormat, errors.New(NotSupported), task.Kind)
	}
	return s.unlinkFileData(string(task.Payload))
}

func (s *Daemon) executeObjectStorageWrite(task *Task) error {
	if s.objectStorage == nil {
		return nil // With no object storage configured, queued tasks finish without doing anything
	}
	switch task.Kind {
	case TaskPutFileData:
		return s.uploadFileData(string(task.Payload))
	case TaskDeleteFileDataInObjectStorage:
		return s.deleteFileDataInObjectStorage(string(task.Payload))
	case TaskPutIndexArchive:
		return s.putIndexArchive()
	}
	return fmt.Errorf(objectStorageWriteUnsupportedKindFormat, errors.New(NotSupported), task.Kind)
}

func isPermanentPhysicalError(err error) bool {
	return Has(err, InvalidArgument) || Has(err, NotSupported)
}

func (s *Daemon) startTaskQueues() {
	s.fileDataDelete = &taskQueue{
		name: FileDataDeleteQueue, workers: s.config.FileDataDeleteWorkers, retry: s.config.FileDataDeleteRetry,
		execute: s.executeFileDataDelete, isPermanent: isPermanentPhysicalError,
	}
	s.objectStorageWrite = &taskQueue{
		name: ObjectStorageWriteQueue, workers: s.config.ObjectStorageWriteWorkers, retry: s.config.ObjectStorageWriteRetry,
		execute: s.executeObjectStorageWrite, isPermanent: isPermanentPhysicalError,
	}
	for _, queue := range s.queues() {
		queue.database = s.index
		queue.logger = s.logger
		queue.idleWait = s.config.WorkerIdleWait
		queue.errorWait = s.config.WorkerErrorWait
		queue.start(s.stop)
	}
}

func (s *Daemon) queues() []*taskQueue {
	return []*taskQueue{s.fileDataDelete, s.objectStorageWrite}
}

func (s *Daemon) queue(name string) *taskQueue {
	for _, queue := range s.queues() {
		if queue.name == name {
			return queue
		}
	}
	return nil
}

func (s *Daemon) waitTaskQueues() {
	for _, queue := range s.queues() {
		queue.wait()
	}
}

func (s *Daemon) update(write func(transaction *sql.Tx) error) error {
	return s.updateWithTasks(func(transaction *sql.Tx) (TaskRequests, error) {
		return nil, write(transaction)
	})
}

func (s *Daemon) updateWithTasks(write func(transaction *sql.Tx) (TaskRequests, error)) error {
	if err := s.checkOpen(); err != nil {
		return err
	}
	// The index change and the tasks it needs are committed together
	var touched []string
	err := s.index.update(func(transaction *sql.Tx) error {
		touched = nil
		if err := s.failurePoint(failurePointIndexWriteBeforeCommit); err != nil {
			return err
		}
		requests, err := write(transaction)
		if err != nil {
			return err
		}
		for _, name := range queueNames {
			for _, request := range requests[name] {
				if _, err := enqueueTask(transaction, name, request); err != nil {
					return err
				}
			}
			if len(requests[name]) > 0 {
				touched = append(touched, name)
			}
		}
		return putMeta(transaction, metaKeyLastWriteAt, time.Now().UnixNano())
	})
	if err != nil {
		return err
	}
	for _, name := range touched {
		s.queue(name).notify()
	}
	return nil
}

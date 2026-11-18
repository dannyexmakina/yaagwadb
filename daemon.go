package yaagwadb

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"

	objectstorage "yaagwadb/object_storage"
)

const (
	daemonErrorFormatInodeCause     = "%w: inode %d: %v" // appends the underlying error as a string
	daemonErrorFormatNegativeOffset = "%w: negative offset"
	daemonErrorFormatNameInParent   = "%w: %q in inode %d" // name not found under the parent inode
	daemonErrorFormatLegacyIndex    = "%w: the data directory has %s of an older version (bbolt), which this version does not read"
)

const (
	failurePointWriteBeforeEnqueue = "write:before-enqueue" // after placing the file data, before registering it in the index
)

const (
	// noAttributeChange means the Setattr mask specifies no change. In that case the index is not written.
	noAttributeChange AttributeMask = 0
	// allTaskQueues, as the queue argument of ListTasks, targets all queues.
	allTaskQueues = ""
)

// Config is the daemon configuration. Zero-valued fields are filled from DefaultConfig.
type Config struct {
	DataDirectory string
	// DatabaseFileName is the name of the SQLite file in DataDirectory holding the index and the two task queues.
	DatabaseFileName string
	// ObjectStorage disables object storage integration if nil.
	ObjectStorage       objectstorage.ObjectStorage
	ObjectStoragePrefix string

	FileDataDeleteWorkers     int
	ObjectStorageWriteWorkers int

	FileDataDeleteRetry     RetryPolicy
	ObjectStorageWriteRetry RetryPolicy

	// SchedulerTick is how often the scheduler checks whether a job is due. Each job interval must be at least this.
	SchedulerTick    time.Duration
	DisableScheduler bool
	// PhysicalDeleteInterval is how often logically deleted file data are found and physical deletion tasks enqueued.
	PhysicalDeleteInterval time.Duration
	// IndexArchiveInterval is how often an index archive (PUT to object storage) is enqueued.
	IndexArchiveInterval time.Duration
	// FinishedTaskCleanupInterval is how often finished tasks past their retention period are removed.
	FinishedTaskCleanupInterval time.Duration
	// LogicalDeleteGracePeriod is the grace period between logical and physical deletion (window to recover mistaken deletions).
	// With object storage it must be longer than IndexArchiveInterval; CheckConfig rejects the configuration otherwise.
	LogicalDeleteGracePeriod time.Duration
	// FailedTaskRetryInterval is how often Failed file data deletion and object storage write tasks are retried.
	FailedTaskRetryInterval time.Duration
	FinishedTaskRetention   time.Duration

	// WorkerIdleWait is the maximum time a worker waits before checking again if no task is runnable.
	// Enqueuing a task wakes it up early.
	WorkerIdleWait time.Duration
	// WorkerErrorWait is how long a worker waits before retrying if dequeuing a task fails.
	WorkerErrorWait time.Duration
	// RunningTaskRecoveryInterval is how often tasks left running past RunningTaskTimeout are returned to pending.
	RunningTaskRecoveryInterval time.Duration
	// RunningTaskTimeout is how long a task may stay running before it is taken as left by a worker that stopped.
	// It must be longer than any task takes.
	RunningTaskTimeout time.Duration
	// DatabaseBusyTimeout is the maximum wait for a lock of an SQLite file.
	DatabaseBusyTimeout time.Duration

	// QueryTimeout is the maximum time one read SQL may run.
	QueryTimeout time.Duration
	// QueryRowLimit is the maximum number of rows one read SQL returns. A statement without LIMIT returns up to this many.
	QueryRowLimit int

	Logger *slog.Logger

	// FailurePoint is for crash injection tests. Returning an error at a named failure point aborts processing there.
	FailurePoint func(name string) error
}

const (
	defaultFileDataDeleteWorkers       = 2
	defaultObjectStorageWriteWorkers   = 8
	defaultSchedulerTick               = 10 * time.Second
	defaultPhysicalDeleteInterval      = 10 * time.Second
	defaultIndexArchiveInterval        = 15 * time.Minute
	defaultFinishedTaskCleanupInterval = time.Hour
	defaultLogicalDeleteGracePeriod    = 24 * time.Hour
	defaultFailedTaskRetryInterval     = time.Hour
	defaultFinishedTaskRetention       = 24 * time.Hour
	defaultWorkerIdleWait              = time.Minute
	defaultWorkerErrorWait             = time.Second
	defaultRunningTaskRecoveryInterval = 10 * time.Minute
	defaultRunningTaskTimeout          = time.Hour
	defaultDatabaseBusyTimeout         = 5 * time.Second
	defaultDatabaseFileName            = "yaagwa.db"
	defaultQueryTimeout                = 10 * time.Second
	defaultQueryRowLimit               = 1000
)

const (
	// defaultRetryJitter is the jitter ratio applied to retry intervals. Shared by all queues.
	defaultRetryJitter = 0.2

	// File data deletion waits out transient local file system failures.
	defaultFileDataDeleteRetryBase        = time.Second
	defaultFileDataDeleteRetryCap         = time.Minute
	defaultFileDataDeleteRetryMaxAttempts = 10
	defaultFileDataDeleteRetryMaxDuration = time.Hour

	// Object storage writes wait longer for network and external service outages.
	defaultObjectStorageWriteRetryBase        = time.Second
	defaultObjectStorageWriteRetryCap         = 10 * time.Minute
	defaultObjectStorageWriteRetryMaxAttempts = 20
	defaultObjectStorageWriteRetryMaxDuration = 6 * time.Hour
)

func DefaultConfig() Config {
	return Config{
		DatabaseFileName:          defaultDatabaseFileName,
		FileDataDeleteWorkers:     defaultFileDataDeleteWorkers,
		ObjectStorageWriteWorkers: defaultObjectStorageWriteWorkers,
		FileDataDeleteRetry: RetryPolicy{Base: defaultFileDataDeleteRetryBase, Cap: defaultFileDataDeleteRetryCap,
			Jitter: defaultRetryJitter, MaxAttempts: defaultFileDataDeleteRetryMaxAttempts, MaxDuration: defaultFileDataDeleteRetryMaxDuration},
		ObjectStorageWriteRetry: RetryPolicy{Base: defaultObjectStorageWriteRetryBase, Cap: defaultObjectStorageWriteRetryCap,
			Jitter: defaultRetryJitter, MaxAttempts: defaultObjectStorageWriteRetryMaxAttempts, MaxDuration: defaultObjectStorageWriteRetryMaxDuration},
		SchedulerTick:               defaultSchedulerTick,
		PhysicalDeleteInterval:      defaultPhysicalDeleteInterval,
		IndexArchiveInterval:        defaultIndexArchiveInterval,
		FinishedTaskCleanupInterval: defaultFinishedTaskCleanupInterval,
		LogicalDeleteGracePeriod:    defaultLogicalDeleteGracePeriod,
		FailedTaskRetryInterval:     defaultFailedTaskRetryInterval,
		FinishedTaskRetention:       defaultFinishedTaskRetention,
		WorkerIdleWait:              defaultWorkerIdleWait,
		WorkerErrorWait:             defaultWorkerErrorWait,
		RunningTaskRecoveryInterval: defaultRunningTaskRecoveryInterval,
		RunningTaskTimeout:          defaultRunningTaskTimeout,
		DatabaseBusyTimeout:         defaultDatabaseBusyTimeout,
		QueryTimeout:                defaultQueryTimeout,
		QueryRowLimit:               defaultQueryRowLimit,
	}
}

func (c Config) withDefaults() Config {
	defaults := DefaultConfig()
	setDefault(&c.DatabaseFileName, defaults.DatabaseFileName)
	setDefault(&c.FileDataDeleteWorkers, defaults.FileDataDeleteWorkers)
	setDefault(&c.ObjectStorageWriteWorkers, defaults.ObjectStorageWriteWorkers)
	c.FileDataDeleteRetry = c.FileDataDeleteRetry.withDefaults(defaults.FileDataDeleteRetry)
	c.ObjectStorageWriteRetry = c.ObjectStorageWriteRetry.withDefaults(defaults.ObjectStorageWriteRetry)
	setDefault(&c.SchedulerTick, defaults.SchedulerTick)
	setDefault(&c.PhysicalDeleteInterval, defaults.PhysicalDeleteInterval)
	setDefault(&c.IndexArchiveInterval, defaults.IndexArchiveInterval)
	setDefault(&c.FinishedTaskCleanupInterval, defaults.FinishedTaskCleanupInterval)
	setDefault(&c.WorkerIdleWait, defaults.WorkerIdleWait)
	setDefault(&c.WorkerErrorWait, defaults.WorkerErrorWait)
	setDefault(&c.RunningTaskRecoveryInterval, defaults.RunningTaskRecoveryInterval)
	setDefault(&c.RunningTaskTimeout, defaults.RunningTaskTimeout)
	setDefault(&c.DatabaseBusyTimeout, defaults.DatabaseBusyTimeout)
	setDefault(&c.QueryTimeout, defaults.QueryTimeout)
	setDefault(&c.QueryRowLimit, defaults.QueryRowLimit)
	setDefault(&c.LogicalDeleteGracePeriod, defaults.LogicalDeleteGracePeriod)
	setDefault(&c.FailedTaskRetryInterval, defaults.FailedTaskRetryInterval)
	setDefault(&c.FinishedTaskRetention, defaults.FinishedTaskRetention)
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
	return c
}

func (p RetryPolicy) withDefaults(defaults RetryPolicy) RetryPolicy {
	setDefault(&p.Base, defaults.Base)
	setDefault(&p.Cap, defaults.Cap)
	setDefault(&p.Jitter, defaults.Jitter)
	setDefault(&p.MaxAttempts, defaults.MaxAttempts)
	setDefault(&p.MaxDuration, defaults.MaxDuration)
	return p
}

func setDefault[T comparable](value *T, fallback T) {
	var zero T
	if *value == zero {
		*value = fallback
	}
}

// Daemon implements the daemon subcommand. It owns the SQLite file (the index and the two task queues),
// the file data, workers, the scheduler, and object storage integration.
type Daemon struct {
	config        Config
	logger        *slog.Logger
	objectStorage objectstorage.ObjectStorage
	index         *database
	// query runs read SQL. Its connections are query_only and carry the authorizer that allows only the objects view.
	query              *sql.DB
	lockFile           *os.File
	fileDataDirectory  string
	temporaryDirectory string

	fileDataDelete     *taskQueue
	objectStorageWrite *taskQueue

	context       context.Context
	cancel        context.CancelFunc
	stop          chan struct{}
	schedulerDone chan struct{}
	closeOnce     sync.Once
}

func Open(config Config) (*Daemon, error) {
	config = config.withDefaults()
	if err := CheckConfig(config); err != nil {
		return nil, err
	}
	daemon := &Daemon{
		config:             config,
		logger:             config.Logger,
		objectStorage:      config.ObjectStorage,
		fileDataDirectory:  filepath.Join(config.DataDirectory, fileDataDirectoryName),
		temporaryDirectory: filepath.Join(config.DataDirectory, temporaryFileDirectoryName),
		stop:               make(chan struct{}),
		schedulerDone:      make(chan struct{}),
	}
	daemon.context, daemon.cancel = context.WithCancel(context.Background())
	if err := daemon.open(); err != nil {
		daemon.release()
		return nil, err
	}
	daemon.startTaskQueues()
	if config.DisableScheduler {
		close(daemon.schedulerDone)
	} else {
		go daemon.runScheduler()
	}
	return daemon, nil
}

func (s *Daemon) open() error {
	for _, directory := range []string{s.config.DataDirectory, s.fileDataDirectory, s.temporaryDirectory} {
		if err := os.MkdirAll(directory, dataDirectoryPermission); err != nil {
			return err
		}
	}
	lockFile, err := lockDataDirectory(s.config.DataDirectory)
	if err != nil {
		return err
	}
	s.lockFile = lockFile
	if err := s.removeTemporaryFiles(); err != nil {
		return err
	}
	// The index of an older version must not be taken as a new, empty one: the startup cleanup would remove all file data
	if _, err := os.Stat(filepath.Join(s.config.DataDirectory, legacyIndexFileName)); err == nil {
		return fmt.Errorf(daemonErrorFormatLegacyIndex, errors.New(NotSupported), legacyIndexFileName)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	indexPath := filepath.Join(s.config.DataDirectory, s.config.DatabaseFileName)
	if err := s.restoreIndexIfMissing(indexPath); err != nil {
		return err
	}
	if s.index, err = openDatabase(indexPath, s.config.DatabaseBusyTimeout); err != nil {
		return err
	}
	if err := initializeIndex(s.index); err != nil {
		return err
	}
	if s.query, err = openQueryDatabase(indexPath, s.config.DatabaseBusyTimeout); err != nil {
		return err
	}
	if err := initializeTaskQueues(s.index); err != nil {
		return err
	}
	// No worker runs yet, so every running task was left by the previous process
	if err := s.index.update(func(transaction *sql.Tx) error {
		for _, name := range queueNames {
			if _, err := resetRunningTasks(transaction, name, time.Now().UnixNano()); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return s.removeUnregisteredFileData()
}

func (s *Daemon) Close() error {
	var err error
	s.closeOnce.Do(func() {
		close(s.stop)
		s.waitTaskQueues()
		<-s.schedulerDone
		err = s.release()
	})
	return err
}

func (s *Daemon) release() error {
	s.cancel()
	errs := []error{s.index.close()}
	if s.query != nil {
		errs = append(errs, s.query.Close())
	}
	if s.lockFile != nil {
		errs = append(errs, s.lockFile.Close())
	}
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Daemon) checkOpen() error {
	select {
	case <-s.stop:
		return errors.New(Closed)
	default:
		return nil
	}
}

func (s *Daemon) failurePoint(name string) error {
	if s.config.FailurePoint == nil {
		return nil
	}
	return s.config.FailurePoint(name)
}

func (s *Daemon) Write(parent uint64, name string, mode, uid, gid uint32, content io.Reader) (*Inode, error) {
	if err := s.checkOpen(); err != nil {
		return nil, err
	}
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	// reject writes that cannot succeed before writing the content (final check happens in the index registration transaction)
	if err := s.index.view(func(transaction *sql.Tx) error {
		return checkWriteTarget(transaction, parent, name)
	}); err != nil {
		return nil, err
	}
	fileDataID, size, hash, err := s.writeFileData(content)
	if err != nil {
		return nil, err
	}
	if err := s.failurePoint(failurePointWriteBeforeEnqueue); err != nil {
		return nil, err
	}
	var inode *Inode
	err = s.updateWithTasks(func(transaction *sql.Tx) (TaskRequests, error) {
		registration := &fileDataRegistration{Parent: parent, Name: name, Mode: mode, UID: uid, GID: gid,
			FileDataID: fileDataID, Size: size, Hash: hash, ModifyTime: time.Now().UnixNano()}
		var err error
		if inode, err = registerFileData(transaction, registration); err != nil {
			return nil, err
		}
		requests := TaskRequests{}
		if s.objectStorage != nil {
			requests.add(ObjectStorageWriteQueue, putFileDataTask(fileDataID))
		}
		return requests, nil
	})
	if err != nil {
		// a file data not registered in the index is invalid, so remove it (startup cleanup removes it otherwise)
		_ = os.Remove(FileDataPath(s.fileDataDirectory, fileDataID))
		return nil, err
	}
	return inode, nil
}

func checkWriteTarget(transaction querier, parent uint64, name string) error {
	if err := checkParentDirectory(transaction, parent); err != nil {
		return err
	}
	inode, err := lookupDirectoryEntry(transaction, parent, name)
	if err != nil {
		return err
	}
	if inode != nil && inode.IsDirectory() {
		return fmt.Errorf(errorFormatQuotedName, errors.New(IsDirectory), name)
	}
	return nil
}

func (s *Daemon) Truncate(inodeNumber uint64) error {
	inode, err := s.GetAttribute(inodeNumber)
	if err != nil {
		return err
	}
	if inode.IsDirectory() {
		return fmt.Errorf(errorFormatInodeNumber, errors.New(IsDirectory), inodeNumber)
	}
	_, err = s.Write(inode.ParentInodeNumber, inode.Name, inode.Mode, inode.UID, inode.GID, eofReader{})
	return err
}

type eofReader struct{}

func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }

func (s *Daemon) Unlink(parent uint64, name string) error {
	return s.update(func(transaction *sql.Tx) error {
		return logicalDelete(transaction, parent, name)
	})
}

func (s *Daemon) Mkdir(parent uint64, name string, mode, uid, gid uint32) (*Inode, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	var inode *Inode
	err := s.update(func(transaction *sql.Tx) error {
		var err error
		inode, err = makeDirectory(transaction, &directoryCreation{Parent: parent, Name: name, Mode: mode, UID: uid, GID: gid})
		return err
	})
	return inode, err
}

func (s *Daemon) Rmdir(parent uint64, name string) error {
	return s.update(func(transaction *sql.Tx) error {
		return removeDirectory(transaction, parent, name)
	})
}

func (s *Daemon) Rename(oldParent uint64, oldName string, newParent uint64, newName string) error {
	if err := ValidateName(newName); err != nil {
		return err
	}
	return s.update(func(transaction *sql.Tx) error {
		return rename(transaction, &renameTarget{OldParent: oldParent, OldName: oldName, NewParent: newParent, NewName: newName})
	})
}

func (s *Daemon) Setattr(inodeNumber uint64, mask AttributeMask, attribute Attribute) (*Inode, error) {
	if mask == noAttributeChange {
		return s.GetAttribute(inodeNumber)
	}
	var inode *Inode
	err := s.update(func(transaction *sql.Tx) error {
		var err error
		inode, err = setAttribute(transaction, &attributeChange{InodeNumber: inodeNumber, Mask: mask, Attribute: attribute})
		return err
	})
	return inode, err
}

func (s *Daemon) SetExtendedAttribute(inodeNumber uint64, name string, value []byte) error {
	if err := validateExtendedAttribute(name, value); err != nil {
		return err
	}
	return s.update(func(transaction *sql.Tx) error {
		if _, err := getInode(transaction, inodeNumber); err != nil {
			return err
		}
		return putExtendedAttribute(transaction, inodeNumber, name, value)
	})
}

func (s *Daemon) RemoveExtendedAttribute(inodeNumber uint64, name string) error {
	return s.update(func(transaction *sql.Tx) error {
		if _, err := getExtendedAttribute(transaction, inodeNumber, name); err != nil {
			return err
		}
		return deleteExtendedAttribute(transaction, inodeNumber, name)
	})
}

func (s *Daemon) view(read func(transaction *sql.Tx) error) error {
	if err := s.checkOpen(); err != nil {
		return err
	}
	return s.index.view(read)
}

func (s *Daemon) Lookup(parent uint64, name string) (*Inode, error) {
	var inode *Inode
	err := s.view(func(transaction *sql.Tx) error {
		var err error
		inode, err = lookup(transaction, parent, name)
		return err
	})
	return inode, err
}

func (s *Daemon) GetAttribute(inodeNumber uint64) (*Inode, error) {
	var inode *Inode
	err := s.view(func(transaction *sql.Tx) error {
		var err error
		inode, err = getInode(transaction, inodeNumber)
		return err
	})
	return inode, err
}

func (s *Daemon) ListDirectory(inodeNumber uint64) ([]DirectoryEntry, error) {
	var entries []DirectoryEntry
	err := s.view(func(transaction *sql.Tx) error {
		if err := checkParentDirectory(transaction, inodeNumber); err != nil {
			return err
		}
		var err error
		entries, err = listDirectory(transaction, inodeNumber)
		return err
	})
	return entries, err
}

func (s *Daemon) OpenFileData(inodeNumber uint64) (*os.File, error) {
	var fileDataID string
	err := s.view(func(transaction *sql.Tx) error {
		inode, err := getInode(transaction, inodeNumber)
		if err != nil {
			return err
		}
		if inode.IsDirectory() {
			return fmt.Errorf(errorFormatInodeNumber, errors.New(IsDirectory), inodeNumber)
		}
		fileData, err := getFileData(transaction, inode.FileDataID)
		if err != nil {
			return fmt.Errorf(daemonErrorFormatInodeCause, errors.New(Corrupted), inodeNumber, err)
		}
		if fileData.State == FileDataLogicallyDeleted {
			return fmt.Errorf(errorFormatInodeNumber, errors.New(NotFound), inodeNumber)
		}
		fileDataID = fileData.FileDataID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.openFileData(fileDataID)
}

func (s *Daemon) ReadAt(inodeNumber uint64, p []byte, offset int64) (int, error) {
	if offset < 0 {
		return 0, fmt.Errorf(daemonErrorFormatNegativeOffset, errors.New(InvalidArgument))
	}
	file, err := s.OpenFileData(inodeNumber)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return file.ReadAt(p, offset)
}

func (s *Daemon) GetExtendedAttribute(inodeNumber uint64, name string) ([]byte, error) {
	var value []byte
	err := s.view(func(transaction *sql.Tx) error {
		if _, err := getInode(transaction, inodeNumber); err != nil {
			return err
		}
		var err error
		value, err = getExtendedAttribute(transaction, inodeNumber, name)
		return err
	})
	return value, err
}

func (s *Daemon) ListExtendedAttributes(inodeNumber uint64) ([]string, error) {
	var names []string
	err := s.view(func(transaction *sql.Tx) error {
		if _, err := getInode(transaction, inodeNumber); err != nil {
			return err
		}
		var err error
		names, err = listExtendedAttributes(transaction, inodeNumber)
		return err
	})
	return names, err
}

// statfsFilesFree is the free inode count returned by Statfs. Inode numbers are 64-bit so there is no practical limit,
// but the value must not overflow for clients using 32-bit statfs.
const statfsFilesFree = 1 << 32

// FileSystemStatus is the result of Statfs.
type FileSystemStatus struct {
	BlockSize, Blocks, BlocksFree, BlocksAvailable uint64
	Files, FilesFree                               uint64
}

func (s *Daemon) Statfs() (*FileSystemStatus, error) {
	var system syscall.Statfs_t
	if err := syscall.Statfs(s.config.DataDirectory, &system); err != nil {
		return nil, err
	}
	status := &FileSystemStatus{
		BlockSize: uint64(system.Bsize), Blocks: uint64(system.Blocks),
		BlocksFree: uint64(system.Bfree), BlocksAvailable: uint64(system.Bavail), FilesFree: statfsFilesFree,
	}
	err := s.view(func(transaction *sql.Tx) error {
		var err error
		status.Files, err = countInodes(transaction)
		return err
	})
	return status, err
}

// QueueStatus holds the counts of one task queue.
type QueueStatus struct {
	Queue                    string
	Pending, Running, Failed uint64
}

// Status summarizes the daemon state.
type Status struct {
	Queues           []QueueStatus
	FileData         uint64
	LogicallyDeleted uint64
}

func (s *Daemon) Status() (*Status, error) {
	if err := s.checkOpen(); err != nil {
		return nil, err
	}
	status := &Status{}
	err := s.view(func(transaction *sql.Tx) error {
		for _, name := range queueNames {
			queue := QueueStatus{Queue: name}
			counts := []struct {
				state TaskState
				count *uint64
			}{{TaskPending, &queue.Pending}, {TaskRunning, &queue.Running}, {TaskFailed, &queue.Failed}}
			for _, target := range counts {
				var err error
				if *target.count, err = countTasks(transaction, name, target.state); err != nil {
					return err
				}
			}
			status.Queues = append(status.Queues, queue)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	err = s.view(func(transaction *sql.Tx) error {
		var err error
		if status.FileData, err = countFileData(transaction, FileDataSucceeded); err != nil {
			return err
		}
		status.LogicallyDeleted, err = countFileData(transaction, FileDataLogicallyDeleted)
		return err
	})
	return status, err
}

// QueuedTask is one ListTasks result.
type QueuedTask struct {
	Queue string
	Task
}

func (s *Daemon) ListTasks(queue string, activeOnly bool, limit int) ([]QueuedTask, error) {
	if err := s.checkOpen(); err != nil {
		return nil, err
	}
	var tasks []QueuedTask
	err := s.index.view(func(transaction *sql.Tx) error {
		for _, name := range queueNames {
			if queue != allTaskQueues && queue != name {
				continue
			}
			found, err := scanTasks(transaction.Query(`SELECT `+taskColumns+` FROM `+name+`
				WHERE NOT ? OR state IN (?, ?) ORDER BY id DESC LIMIT ?`, activeOnly, TaskPending, TaskRunning, limit))
			if err != nil {
				return err
			}
			for _, task := range found {
				tasks = append(tasks, QueuedTask{Queue: name, Task: *task})
			}
		}
		return nil
	})
	return tasks, err
}

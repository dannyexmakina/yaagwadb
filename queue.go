package yaagwadb

import (
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	FileDataDeleteQueue     = "file_data_delete_task_queue"
	ObjectStorageWriteQueue = "object_storage_write_task_queue"
)

const (
	taskStatePendingName   = "pending"
	taskStateRunningName   = "running"
	taskStateSucceededName = "succeeded"
	taskStateFailedName    = "failed"
	taskStateUnknownFormat = "TaskState(%d)"
)

const (
	taskWithoutKey   = ""
	taskWithoutError = ""
)

// emptyPayload is the payload of a task that needs none.
var emptyPayload = []byte{}

// taskNotFinished, as FinishedAt, means the task has not finished yet. taskNotStarted, as StartedAt, means it has not run yet.
const (
	taskNotFinished = 0
	taskNotStarted  = 0
)

// workerGoroutineStep is added to the WaitGroup for each worker goroutine started.
const workerGoroutineStep = 1

const (
	retryFirstAttempt       = 1
	retryBackoffMultiplier  = 2
	retryJitterRandomScale  = 2
	retryJitterRandomOffset = 1
)

const (
	queueTaskNotFoundFormat = "%w: task %d in %s"
)

const (
	taskLogTakeFailed          = "take task"
	taskLogSucceeded           = "task succeeded"
	taskLogFailed              = "task failed"
	taskLogRecordFailureFailed = "record task failure"
	taskLogKeyQueue            = "queue"
	taskLogKeyTaskID           = "taskID"
	taskLogKeyKind             = "kind"
	taskLogKeyAttempt          = "attempt"
	taskLogKeyError            = "error"
	taskLogKeyStack            = "stack"
)

var queueNames = []string{FileDataDeleteQueue, ObjectStorageWriteQueue}

// taskQueueSchemaFormat creates the table of one queue. %[1]s is the queue name, %[2]d and %[3]d the unfinished states.
// A key is unique among the unfinished tasks of a queue.
var taskQueueSchemaFormat = []string{
	`CREATE TABLE IF NOT EXISTS %[1]s (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		kind        INTEGER NOT NULL,
		state       INTEGER NOT NULL,
		key         TEXT    NOT NULL,
		payload     BLOB    NOT NULL,
		attempts    INTEGER NOT NULL,
		next_run_at INTEGER NOT NULL,
		last_error  TEXT    NOT NULL,
		created_at  INTEGER NOT NULL,
		started_at  INTEGER NOT NULL,
		finished_at INTEGER NOT NULL
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS %[1]s_active_key ON %[1]s (key) WHERE key != '' AND state IN (%[2]d, %[3]d)`,
	`CREATE INDEX IF NOT EXISTS %[1]s_ready ON %[1]s (state, next_run_at)`,
}

const taskColumns = `id, kind, state, key, payload, attempts, next_run_at, last_error, created_at, started_at, finished_at`

type TaskState uint8

const (
	TaskPending   TaskState = 0
	TaskRunning   TaskState = 1
	TaskSucceeded TaskState = 2
	TaskFailed    TaskState = 3
)

func (s TaskState) String() string {
	switch s {
	case TaskPending:
		return taskStatePendingName
	case TaskRunning:
		return taskStateRunningName
	case TaskSucceeded:
		return taskStateSucceededName
	case TaskFailed:
		return taskStateFailedName
	}
	return fmt.Sprintf(taskStateUnknownFormat, s)
}

// TaskKind is the task type. Values are defined per queue (task.go).
type TaskKind uint8

// Task is one entry in a task queue.
type Task struct {
	ID         uint64
	Kind       TaskKind
	State      TaskState
	Key        string // if non-empty, unique among the unfinished tasks of the queue
	Payload    []byte
	Attempts   uint32
	NextRunAt  int64
	LastError  string
	CreatedAt  int64
	StartedAt  int64
	FinishedAt int64
}

func (t *Task) active() bool { return t.State == TaskPending || t.State == TaskRunning }

func scanTask(row scanner) (*Task, error) {
	task := &Task{}
	err := row.Scan(&task.ID, &task.Kind, &task.State, &task.Key, &task.Payload, &task.Attempts, &task.NextRunAt,
		&task.LastError, &task.CreatedAt, &task.StartedAt, &task.FinishedAt)
	return task, err
}

func scanTasks(rows *sql.Rows, err error) ([]*Task, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tasks []*Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

// RetryPolicy controls retries (exponential backoff and cap). Configured separately per task type.
type RetryPolicy struct {
	Base        time.Duration
	Cap         time.Duration
	Jitter      float64 // 0.2 means ±20%
	MaxAttempts uint32
	MaxDuration time.Duration
}

func (p RetryPolicy) delay(attempts uint32) time.Duration {
	delay := p.Base
	for attempt := uint32(retryFirstAttempt); attempt < attempts; attempt++ {
		delay *= retryBackoffMultiplier
		if delay >= p.Cap {
			delay = p.Cap
			break
		}
	}
	jitter := (rand.Float64()*retryJitterRandomScale - retryJitterRandomOffset) * p.Jitter * float64(delay)
	return delay + time.Duration(jitter)
}

func initializeTaskQueues(tasks *database) error {
	return tasks.update(func(transaction *sql.Tx) error {
		for _, name := range queueNames {
			for _, format := range taskQueueSchemaFormat {
				if _, err := transaction.Exec(fmt.Sprintf(format, name, TaskPending, TaskRunning)); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func getTask(transaction querier, name string, taskID uint64) (*Task, error) {
	task, err := scanTask(transaction.QueryRow(`SELECT `+taskColumns+` FROM `+name+` WHERE id = ?`, taskID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(queueTaskNotFoundFormat, errors.New(NotFound), taskID, name)
	}
	return task, err
}

func putTask(transaction querier, name string, task *Task) error {
	if task.Payload == nil {
		task.Payload = emptyPayload // the database gives an empty BLOB back as nil, and payload is NOT NULL
	}
	_, err := transaction.Exec(`UPDATE `+name+` SET kind = ?, state = ?, key = ?, payload = ?, attempts = ?, next_run_at = ?,
		last_error = ?, created_at = ?, started_at = ?, finished_at = ? WHERE id = ?`,
		task.Kind, task.State, task.Key, task.Payload, task.Attempts, task.NextRunAt,
		task.LastError, task.CreatedAt, task.StartedAt, task.FinishedAt, task.ID)
	return err
}

func hasActiveTask(transaction querier, name, key string) (bool, error) {
	var exists bool
	err := transaction.QueryRow(`SELECT EXISTS (SELECT 1 FROM `+name+` WHERE key = ? AND state IN (?, ?))`,
		key, TaskPending, TaskRunning).Scan(&exists)
	return exists, err
}

// TaskRequest is the content of a task passed to enqueueTask.
// A task with a Key is not enqueued while an unfinished task with the same Key exists.
type TaskRequest struct {
	Kind    TaskKind
	Key     string
	Payload []byte
}

func enqueueTask(transaction querier, name string, request TaskRequest) (*Task, error) {
	if request.Key != taskWithoutKey {
		exists, err := hasActiveTask(transaction, name, request.Key)
		if err != nil || exists {
			return nil, err
		}
	}
	now := time.Now().UnixNano()
	task := &Task{Kind: request.Kind, State: TaskPending, Key: request.Key, Payload: request.Payload,
		NextRunAt: now, LastError: taskWithoutError, CreatedAt: now}
	if request.Payload == nil {
		task.Payload = emptyPayload
	}
	result, err := transaction.Exec(`INSERT INTO `+name+` (kind, state, key, payload, attempts, next_run_at, last_error,
		created_at, started_at, finished_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.Kind, task.State, task.Key, task.Payload, task.Attempts, task.NextRunAt, task.LastError,
		task.CreatedAt, taskNotStarted, taskNotFinished)
	if err != nil {
		return nil, err
	}
	taskID, err := result.LastInsertId()
	task.ID = uint64(taskID)
	return task, err
}

func finishTask(transaction querier, name string, task *Task, state TaskState, lastError string) error {
	task.State = state
	task.LastError = lastError
	task.FinishedAt = time.Now().UnixNano()
	return putTask(transaction, name, task)
}

func reviveTask(transaction querier, name string, task *Task) error {
	if task.Key != taskWithoutKey {
		exists, err := hasActiveTask(transaction, name, task.Key)
		if err != nil || exists {
			return err // a later task for the same target exists
		}
	}
	task.State = TaskPending
	task.Attempts = 0
	task.CreatedAt = time.Now().UnixNano()
	task.NextRunAt = task.CreatedAt
	task.StartedAt = taskNotStarted
	task.FinishedAt = taskNotFinished
	return putTask(transaction, name, task)
}

func resetRunningTasks(transaction querier, name string, startedBefore int64) (int64, error) {
	result, err := transaction.Exec(`UPDATE `+name+` SET state = ?, next_run_at = ? WHERE state = ? AND started_at <= ?`,
		TaskPending, time.Now().UnixNano(), TaskRunning, startedBefore)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func listTasksInState(transaction querier, name string, state TaskState) ([]*Task, error) {
	return scanTasks(transaction.Query(`SELECT `+taskColumns+` FROM `+name+` WHERE state = ? ORDER BY id`, state))
}

func listActiveTaskPayloads(transaction querier, name string) ([][]byte, error) {
	rows, err := transaction.Query(`SELECT payload FROM `+name+` WHERE state IN (?, ?)`, TaskPending, TaskRunning)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var payloads [][]byte
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}
	return payloads, rows.Err()
}

func countTasks(transaction querier, name string, state TaskState) (uint64, error) {
	var count uint64
	err := transaction.QueryRow(`SELECT COUNT(*) FROM `+name+` WHERE state = ?`, state).Scan(&count)
	return count, err
}

func deleteFinishedTasks(transaction querier, name string, state TaskState, finishedBefore int64) error {
	_, err := transaction.Exec(`DELETE FROM `+name+` WHERE state = ? AND finished_at != ? AND finished_at < ?`,
		state, taskNotFinished, finishedBefore)
	return err
}

// taskQueue is one task queue and the workers that process it.
type taskQueue struct {
	name     string
	database *database
	logger   *slog.Logger
	retry    RetryPolicy
	workers  int
	// execute processes one task.
	execute func(*Task) error
	// isPermanent reports whether the error would not change on retry.
	isPermanent func(error) bool
	// idleWait is the maximum wait when no task is runnable; errorWait is the wait after a failed dequeue.
	idleWait  time.Duration
	errorWait time.Duration

	wake chan struct{}
	stop chan struct{}
	done sync.WaitGroup
}

func (q *taskQueue) start(stop chan struct{}) {
	q.wake = make(chan struct{}, q.workers)
	q.stop = stop
	for range q.workers {
		q.done.Add(workerGoroutineStep)
		go q.run()
	}
}

func (q *taskQueue) wait() { q.done.Wait() }

func (q *taskQueue) notify() {
	for range q.workers {
		select {
		case q.wake <- struct{}{}:
		default:
			return
		}
	}
}

func (q *taskQueue) run() {
	defer q.done.Done()
	for {
		select {
		case <-q.stop:
			return
		default:
		}
		task, wait, err := q.take()
		if err != nil {
			q.logger.Error(taskLogTakeFailed, taskLogKeyQueue, q.name, taskLogKeyError, err)
			wait = q.errorWait
		}
		if task == nil {
			timer := time.NewTimer(wait)
			select {
			case <-q.stop:
				timer.Stop()
				return
			case <-q.wake:
			case <-timer.C:
			}
			timer.Stop()
			continue
		}
		q.process(task)
	}
}

func (q *taskQueue) take() (*Task, time.Duration, error) {
	var task *Task
	wait := q.idleWait
	err := q.database.update(func(transaction *sql.Tx) error {
		now := time.Now().UnixNano()
		found, err := scanTask(transaction.QueryRow(`UPDATE `+q.name+` SET state = ?, attempts = attempts + 1, started_at = ?
			WHERE id = (SELECT id FROM `+q.name+` WHERE state = ? AND next_run_at <= ? ORDER BY next_run_at, id LIMIT 1)
			RETURNING `+taskColumns, TaskRunning, now, TaskPending, now))
		if err == nil {
			task = found
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var nextRunAt sql.NullInt64
		if err := transaction.QueryRow(`SELECT MIN(next_run_at) FROM `+q.name+` WHERE state = ?`, TaskPending).Scan(&nextRunAt); err != nil {
			return err
		}
		if nextRunAt.Valid && nextRunAt.Int64 > now {
			wait = min(wait, time.Duration(nextRunAt.Int64-now))
		}
		return nil
	})
	return task, wait, err
}

func (q *taskQueue) process(task *Task) {
	logger := q.logger.With(taskLogKeyQueue, q.name, taskLogKeyTaskID, task.ID, taskLogKeyKind, task.Kind, taskLogKeyAttempt, task.Attempts)
	err := q.execute(task)
	if err == nil {
		err = q.database.update(func(transaction *sql.Tx) error {
			return finishTask(transaction, q.name, task, TaskSucceeded, taskWithoutError)
		})
	}
	if err == nil {
		logger.Debug(taskLogSucceeded)
		return
	}
	logger.Warn(taskLogFailed, taskLogKeyError, err, taskLogKeyStack, StackOf(err))
	if failure := q.fail(task, err); failure != nil {
		logger.Error(taskLogRecordFailureFailed, taskLogKeyError, failure)
	}
}

func (q *taskQueue) fail(task *Task, cause error) error {
	elapsed := time.Duration(time.Now().UnixNano() - task.CreatedAt)
	permanent := q.isPermanent != nil && q.isPermanent(cause)
	final := permanent || task.Attempts >= q.retry.MaxAttempts || elapsed >= q.retry.MaxDuration
	return q.database.update(func(transaction *sql.Tx) error {
		if !final {
			task.State = TaskPending
			task.LastError = cause.Error()
			task.NextRunAt = time.Now().Add(q.retry.delay(task.Attempts)).UnixNano()
			return putTask(transaction, q.name, task)
		}
		return finishTask(transaction, q.name, task, TaskFailed, cause.Error())
	})
}

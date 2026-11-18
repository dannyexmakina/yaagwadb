package yaagwadb

import (
	"database/sql"
	"time"
)

type scheduledJob struct {
	name     string
	interval time.Duration
	run      func() error
}

const (
	jobPhysicalDelete      = "physical delete"
	jobIndexArchive        = "index archive"
	jobRetryFailedTasks    = "retry failed tasks"
	jobRemoveFinishedTasks = "remove finished tasks"
	jobRecoverRunningTasks = "recover running tasks"
)

const (
	schedulerLogJobFailed = "scheduled job failed"
	schedulerLogKeyJob    = "job"
)

func (s *Daemon) scheduledJobs() []scheduledJob {
	return []scheduledJob{
		{jobPhysicalDelete, s.config.PhysicalDeleteInterval, s.schedulePhysicalDelete},
		{jobIndexArchive, s.config.IndexArchiveInterval, s.scheduleIndexArchive},
		{jobRetryFailedTasks, s.config.FailedTaskRetryInterval, s.retryFailedTasks},
		{jobRecoverRunningTasks, s.config.RunningTaskRecoveryInterval, s.recoverRunningTasks},
		{jobRemoveFinishedTasks, s.config.FinishedTaskCleanupInterval, s.removeFinishedTasks},
	}
}

func (s *Daemon) runScheduler() {
	defer close(s.schedulerDone)
	ticker := time.NewTicker(s.config.SchedulerTick)
	defer ticker.Stop()
	lastRun := map[string]time.Time{}
	for {
		select {
		case <-s.stop:
			return
		case now := <-ticker.C:
			for _, job := range s.scheduledJobs() {
				if now.Sub(lastRun[job.name]) < job.interval {
					continue
				}
				lastRun[job.name] = now
				if err := job.run(); err != nil {
					s.logger.Warn(schedulerLogJobFailed, schedulerLogKeyJob, job.name, taskLogKeyError, err)
				}
			}
		}
	}
}

func (s *Daemon) RunScheduledJobs() error {
	for _, job := range s.scheduledJobs() {
		if err := job.run(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Daemon) schedulePhysicalDelete() error {
	threshold := time.Now().Add(-s.config.LogicalDeleteGracePeriod).UnixNano()
	return s.updateWithTasks(func(transaction *sql.Tx) (TaskRequests, error) {
		expired, err := listExpiredFileData(transaction, threshold)
		if err != nil {
			return nil, err
		}
		requests := TaskRequests{}
		for _, fileDataID := range expired {
			if err := deleteFileData(transaction, fileDataID); err != nil {
				return nil, err
			}
			requests.add(FileDataDeleteQueue, fileDataDeleteTask(fileDataID))
			if s.objectStorage != nil {
				requests.add(ObjectStorageWriteQueue, deleteFileDataInObjectStorageTask(fileDataID))
			}
		}
		return requests, nil
	})
}

func (s *Daemon) scheduleIndexArchive() error {
	if s.objectStorage == nil {
		return nil
	}
	if err := s.checkOpen(); err != nil {
		return err
	}
	// Queuing an archive does not modify the index, so it bypasses update and keeps the last write time
	scheduled := false
	err := s.index.update(func(transaction *sql.Tx) error {
		lastWriteAt, err := getMeta(transaction, metaKeyLastWriteAt)
		if err != nil {
			return err
		}
		var lastArchiveAt sql.NullInt64
		if err := transaction.QueryRow(`SELECT MAX(created_at) FROM `+ObjectStorageWriteQueue+` WHERE kind = ? AND state != ?`,
			TaskPutIndexArchive, TaskFailed).Scan(&lastArchiveAt); err != nil {
			return err
		}
		if lastArchiveAt.Valid && lastWriteAt <= lastArchiveAt.Int64 {
			return nil
		}
		task, err := enqueueTask(transaction, ObjectStorageWriteQueue, indexArchiveTask())
		scheduled = task != nil
		return err
	})
	if scheduled && err == nil {
		s.queue(ObjectStorageWriteQueue).notify()
	}
	return err
}

func (s *Daemon) retryFailedTasks() error {
	if err := s.checkOpen(); err != nil {
		return err
	}
	var revived []string
	err := s.index.update(func(transaction *sql.Tx) error {
		for _, name := range queueNames {
			failed, err := listTasksInState(transaction, name, TaskFailed)
			if err != nil {
				return err
			}
			for _, task := range failed {
				if err := reviveTask(transaction, name, task); err != nil {
					return err
				}
			}
			if len(failed) > 0 {
				revived = append(revived, name)
			}
		}
		return nil
	})
	for _, name := range revived {
		s.queue(name).notify()
	}
	return err
}

func (s *Daemon) recoverRunningTasks() error {
	if err := s.checkOpen(); err != nil {
		return err
	}
	startedBefore := time.Now().Add(-s.config.RunningTaskTimeout).UnixNano()
	var recovered []string
	err := s.index.update(func(transaction *sql.Tx) error {
		for _, name := range queueNames {
			count, err := resetRunningTasks(transaction, name, startedBefore)
			if err != nil {
				return err
			}
			if count > 0 {
				recovered = append(recovered, name)
			}
		}
		return nil
	})
	for _, name := range recovered {
		s.queue(name).notify()
	}
	return err
}

func (s *Daemon) removeFinishedTasks() error {
	if err := s.checkOpen(); err != nil {
		return err
	}
	threshold := time.Now().Add(-s.config.FinishedTaskRetention).UnixNano()
	return s.index.update(func(transaction *sql.Tx) error {
		for _, name := range queueNames {
			if err := deleteFinishedTasks(transaction, name, TaskSucceeded, threshold); err != nil {
				return err
			}
		}
		return nil
	})
}

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"yaagwadb"
	objectstorage "yaagwadb/object_storage"

	"cloud.google.com/go/storage"
	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
)

const (
	defaultConfigPath    = "/etc/yaagwadb/config.toml"
	defaultDataDirectory = "/var/lib/yaagwadb"
	defaultSocket        = "/run/yaagwadb/daemon.sock"
	// defaultHranaListen is on localhost: the SQL front has no authentication of its own
	defaultHranaListen            = "127.0.0.1:8080"
	defaultHranaFileDataSizeLimit = 1 << 20
	defaultHranaFileMode          = 0o644
)

const (
	objectStorageNone = "" // no object storage
	objectStorageS3   = "s3"
	objectStorageGCS  = "gcs"
)

// unsetSetting is the value of a string setting omitted from the config file.
const unsetSetting = ""

// s3UsePathStyle uses path-style bucket addressing for S3-compatible storage with an explicit endpoint.
// Many compatible implementations do not support virtual-hosted style (bucket subdomains).
const s3UsePathStyle = true

const (
	UnknownObjectStorageKind = "yaagwadb: unknown object storage kind"
	ReadConfigFailed         = "yaagwadb: read config"
)

const (
	readConfigFormat               = "%w %s: %w"
	unknownObjectStorageKindFormat = "%w: %q"
)

// fileConfig is the shape of the config file (TOML). Omitted values default to yaagwadb.DefaultConfig.
type fileConfig struct {
	DataDirectory    string `toml:"data_directory"`
	DatabaseFileName string `toml:"database_file_name"`
	Socket           string `toml:"socket"`
	LogLevel         string `toml:"log_level"`

	ObjectStorage struct {
		Kind     string `toml:"kind"` // a constant such as objectStorageS3
		Bucket   string `toml:"bucket"`
		Prefix   string `toml:"prefix"`
		Region   string `toml:"region"`
		Endpoint string `toml:"endpoint"` // endpoint of S3-compatible storage or the GCS emulator
	} `toml:"object_storage"`

	Daemon struct {
		FileDataDeleteWorkers       int           `toml:"file_data_delete_workers"`
		ObjectStorageWriteWorkers   int           `toml:"object_storage_write_workers"`
		SchedulerTick               time.Duration `toml:"scheduler_tick"`
		PhysicalDeleteInterval      time.Duration `toml:"physical_delete_interval"`
		IndexArchiveInterval        time.Duration `toml:"index_archive_interval"`
		FinishedTaskCleanupInterval time.Duration `toml:"finished_task_cleanup_interval"`
		LogicalDeleteGracePeriod    time.Duration `toml:"logical_delete_grace_period"`
		FailedTaskRetryInterval     time.Duration `toml:"failed_task_retry_interval"`
		FinishedTaskRetention       time.Duration `toml:"finished_task_retention"`
		WorkerIdleWait              time.Duration `toml:"worker_idle_wait"`
		WorkerErrorWait             time.Duration `toml:"worker_error_wait"`
		RunningTaskRecoveryInterval time.Duration `toml:"running_task_recovery_interval"`
		RunningTaskTimeout          time.Duration `toml:"running_task_timeout"`
		DatabaseBusyTimeout         time.Duration `toml:"database_busy_timeout"`
		QueryTimeout                time.Duration `toml:"query_timeout"`
		QueryRowLimit               int           `toml:"query_row_limit"`
	} `toml:"daemon"`

	Retry struct {
		FileDataDelete     retryConfig `toml:"file_data_delete"`
		ObjectStorageWrite retryConfig `toml:"object_storage_write"`
	} `toml:"retry"`

	Mount struct {
		Point      string `toml:"point"`
		Debug      bool   `toml:"debug"`
		AllowOther bool   `toml:"allow_other"`
	} `toml:"mount"`

	Hrana struct {
		Listen            string `toml:"listen"`
		FileDataSizeLimit int64  `toml:"file_data_size_limit"`
		FileMode          uint32 `toml:"file_mode"`
	} `toml:"hrana"`
}

type retryConfig struct {
	Base        time.Duration `toml:"base"`
	Cap         time.Duration `toml:"cap"`
	MaxAttempts uint32        `toml:"max_attempts"`
	MaxDuration time.Duration `toml:"max_duration"`
}

func (r retryConfig) policy() yaagwadb.RetryPolicy {
	return yaagwadb.RetryPolicy{Base: r.Base, Cap: r.Cap, MaxAttempts: r.MaxAttempts, MaxDuration: r.MaxDuration}
}

func loadConfig(path string) (*fileConfig, error) {
	if err := checkConfigFile(path); err != nil {
		return nil, err
	}
	configuration := &fileConfig{DataDirectory: defaultDataDirectory, Socket: defaultSocket}
	configuration.Hrana.Listen = defaultHranaListen
	configuration.Hrana.FileDataSizeLimit = defaultHranaFileDataSizeLimit
	configuration.Hrana.FileMode = defaultHranaFileMode
	if _, err := toml.DecodeFile(path, configuration); err != nil {
		return nil, fmt.Errorf(readConfigFormat, errors.New(ReadConfigFailed), path, err)
	}
	return configuration, nil
}

type configLoader func() (*fileConfig, error)

func newLogger(level string) *slog.Logger {
	var parsed slog.Level
	if err := parsed.UnmarshalText([]byte(level)); err != nil {
		parsed = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: parsed}))
}

func newObjectStorage(ctx context.Context, configuration *fileConfig) (objectstorage.ObjectStorage, error) {
	settings := configuration.ObjectStorage
	switch settings.Kind {
	case objectStorageNone:
		return nil, nil
	case objectStorageS3:
		options := []func(*config.LoadOptions) error{}
		if settings.Region != unsetSetting {
			options = append(options, config.WithRegion(settings.Region))
		}
		awsConfig, err := config.LoadDefaultConfig(ctx, options...)
		if err != nil {
			return nil, err
		}
		client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
			if settings.Endpoint != unsetSetting {
				o.BaseEndpoint = &settings.Endpoint
				o.UsePathStyle = s3UsePathStyle
			}
		})
		return objectstorage.NewS3(client, settings.Bucket), nil
	case objectStorageGCS:
		options := []option.ClientOption{}
		if settings.Endpoint != unsetSetting {
			// Authentication is skipped by the client library only when STORAGE_EMULATOR_HOST is set
			options = append(options, option.WithEndpoint(settings.Endpoint))
		}
		client, err := storage.NewClient(ctx, options...)
		if err != nil {
			return nil, err
		}
		return objectstorage.NewGoogleCloudStorage(client, settings.Bucket), nil
	}
	return nil, fmt.Errorf(unknownObjectStorageKindFormat, errors.New(UnknownObjectStorageKind), settings.Kind)
}

func daemonConfig(configuration *fileConfig) (yaagwadb.Config, error) {
	storage, err := newObjectStorage(context.Background(), configuration)
	if err != nil {
		return yaagwadb.Config{}, err
	}
	settings := configuration.Daemon
	return yaagwadb.Config{
		DataDirectory:               configuration.DataDirectory,
		DatabaseFileName:            configuration.DatabaseFileName,
		ObjectStorage:               storage,
		ObjectStoragePrefix:         configuration.ObjectStorage.Prefix,
		FileDataDeleteWorkers:       settings.FileDataDeleteWorkers,
		ObjectStorageWriteWorkers:   settings.ObjectStorageWriteWorkers,
		FileDataDeleteRetry:         configuration.Retry.FileDataDelete.policy(),
		ObjectStorageWriteRetry:     configuration.Retry.ObjectStorageWrite.policy(),
		SchedulerTick:               settings.SchedulerTick,
		PhysicalDeleteInterval:      settings.PhysicalDeleteInterval,
		IndexArchiveInterval:        settings.IndexArchiveInterval,
		FinishedTaskCleanupInterval: settings.FinishedTaskCleanupInterval,
		LogicalDeleteGracePeriod:    settings.LogicalDeleteGracePeriod,
		FailedTaskRetryInterval:     settings.FailedTaskRetryInterval,
		FinishedTaskRetention:       settings.FinishedTaskRetention,
		WorkerIdleWait:              settings.WorkerIdleWait,
		WorkerErrorWait:             settings.WorkerErrorWait,
		RunningTaskRecoveryInterval: settings.RunningTaskRecoveryInterval,
		RunningTaskTimeout:          settings.RunningTaskTimeout,
		DatabaseBusyTimeout:         settings.DatabaseBusyTimeout,
		QueryTimeout:                settings.QueryTimeout,
		QueryRowLimit:               settings.QueryRowLimit,
		Logger:                      newLogger(configuration.LogLevel),
	}, nil
}

# Configuration file

The yaagwadb daemon, mount, and hrana read a TOML configuration file.

## Creating the configuration file

`yaagwadb install` writes an example configuration file to `/etc/yaagwadb/config.toml`. Another location can be specified with `-c`.

- An existing file is not overwritten
- It is written with mode `0640`. The owner is the user who ran the command (usually root)
- The systemd units always read `/etc/yaagwadb/config.toml`. If `-c` writes the file elsewhere, see [systemd.md](systemd.md)

## Loading

- The default location is `/etc/yaagwadb/config.toml`. A file can be specified with `-c` (`--config`)
- Omitted items take the default values in the tables below
- Relative paths are resolved against the directory where the command was run
- Durations are written as strings of numbers and units
  - The units are `ns`, `us`, `ms`, `s`, `m`, and `h`
  - Units can be combined (`"1h30m"` is 1 hour 30 minutes)
  - Decimals can be used (`"1.5h"` is 1 hour 30 minutes)
  - There is no unit for days. Write 7 days as `"168h"`

## Checking

`yaagwadb configtest` checks that the configuration file exists and that its values are valid.

- If there is no problem, it prints `<path of the configuration file>: configuration OK` and exits
- If there is a problem, it prints the error and exits with code 1. See [errors.md](errors.md) for the errors
- Each command also runs checks at startup. What is checked differs by command
  - That the configuration file exists: every command that reads the configuration file (not `install`; `restore` only with `--force`)
  - The daemon values (`kind` in `[object_storage]`, a non-empty `data_directory`, and `logical_delete_grace_period` against `index_archive_interval`): the daemon and `restore`
  - `file_data_size_limit` in `[hrana]`: hrana
  - configtest runs all of the above
- The following are not checked
  - An invalid `log_level`. It is not an error; it is treated as `info`
  - Ranges of numbers (such as a negative number of workers)
  - Whether a `*_interval` is shorter than `scheduler_tick`

## Top level

| Item | Default | Description |
| --- | --- | --- |
| `data_directory` | `"/var/lib/yaagwadb"` | Directory that holds the index and the task queues (`database_file_name`), the file data (`filedata/`), and temporary files (`temporary/`). See [systemd.md](systemd.md) for the limits under systemd |
| `database_file_name` | `"yaagwa.db"` | Name of the SQLite file in `data_directory` that holds the index and the task queues |
| `socket` | `"/run/yaagwadb/daemon.sock"` | Unix domain socket on which the daemon listens for gRPC. See [systemd.md](systemd.md) for the limits under systemd |
| `log_level` | `"info"` | Log level. `debug`, `info`, `warn`, or `error` |

## `[object_storage]`

Settings for the object storage that holds the file data and the backups of the index.

| Item | Default | Description |
| --- | --- | --- |
| `kind` | `""` | Either `"s3"` or `"gcs"`. If empty, no backup is made |
| `bucket` | `""` | Bucket name (`s3`, `gcs`) |
| `prefix` | `""` | Prefix for the paths of the files in object storage |
| `region` | `""` | Region for `s3`. If empty, the AWS SDK's default resolution order is used |
| `endpoint` | `""` | See "Endpoint" below |

### Endpoint

- For `s3`, the endpoint of S3-compatible storage. Buckets are addressed in path style
- For `gcs`, the Cloud Storage endpoint (such as Private Service Connect). If empty, the default endpoint

### Authentication

Credentials are not written in the config file. They are obtained in each SDK's default resolution order.

- `s3`
  - On AWS, an IAM role (EC2 instance profile, ECS task role, EKS IRSA or Pod Identity)
  - Outside AWS, the environment variables `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` (including S3-compatible storage), or a profile in `~/.aws/credentials` (selected with `AWS_PROFILE`)
- `gcs`
  - On Google Cloud, the attached service account (GCE) or Workload Identity (GKE)
  - Outside Google Cloud, a Workload Identity Federation configuration file or a service account key file. Its path is set in the environment variable `GOOGLE_APPLICATION_CREDENTIALS`
  - For an emulator (such as fake-gcs-server), set the host and port in the environment variable `STORAGE_EMULATOR_HOST`. No authentication is used. Leave `endpoint` empty

Set the environment variables in the environment of the daemon and `restore`.

## `[daemon]`

| Item | Default | Description |
| --- | --- | --- |
| `file_data_delete_workers` | `2` | Number of parallel file data delete workers |
| `object_storage_write_workers` | `8` | Number of parallel object storage write workers |
| `scheduler_tick` | `"10s"` | How often periodic jobs are checked. A `*_interval` shorter than this has no effect |
| `physical_delete_interval` | `"10s"` | How often logically deleted file data is physically deleted |
| `index_archive_interval` | `"15m"` | How often the index is backed up to object storage |
| `finished_task_cleanup_interval` | `"1h"` | How often finished tasks are deleted |
| `logical_delete_grace_period` | `"24h"` | Period from logical deletion to physical deletion. With object storage, it must be longer than `index_archive_interval`, or the daemon does not start |
| `failed_task_retry_interval` | `"1h"` | How often tasks that reached their retry limit and ended as Failed are retried |
| `finished_task_retention` | `"24h"` | How long succeeded tasks are kept. After that, they are deleted at every `finished_task_cleanup_interval` |
| `worker_idle_wait` | `"1m"` | If no task is runnable, the longest time until a worker runs next |
| `worker_error_wait` | `"1s"` | If taking a task fails, the time until the worker's next attempt |
| `running_task_recovery_interval` | `"10m"` | How often tasks stuck in `running` are checked |
| `running_task_timeout` | `"1h"` | Time after which a `running` task is returned to `pending`. Longer than any task takes |
| `database_busy_timeout` | `"5s"` | Longest wait for a lock of the `database_file_name` file |
| `query_timeout` | `"10s"` | Longest time one read SQL statement runs |
| `query_row_limit` | `1000` | Most rows one read SQL statement returns. Also the largest LIMIT hrana accepts (hrana uses the value in the configuration file it reads; use the same file as the daemon) |

### What is lost if the local data is lost

If the data directory is lost, the following two are not recovered by restoring from object storage.

- File data whose PUT to object storage had not finished. This includes both pending and running PUTs and PUTs that reached their retry limit and ended as Failed. `yaagwadb status` shows how many there are
- Changes to the index after the last index backup (new files, Rename, logical deletion, changes to attributes or xattrs, and so on). A shorter `index_archive_interval` makes this range smaller

File data written after the last index backup remains as files in object storage that the restored index does not refer to.

### How restoring works

- With object storage configured, if `yaagwa.db` does not exist when the daemon starts, the daemon downloads the latest index from object storage before starting. If the data directory is lost, starting the daemon is enough to restore it
- File data is not downloaded when restoring. File data that is not local is downloaded from object storage one at a time when it is read
- `yaagwadb restore --force` is for broken local data that is still there. It moves `filedata/` and `yaagwa.db` (including `-wal` and `-shm`) to names with `.broken-<UNIX nanoseconds>` appended, restores the index in the same way as above, and exits
  - It locks the data directory, so stop the daemon first
  - File data whose PUT had not finished remains in `filedata.broken-<UNIX nanoseconds>/`
  - An `index.db` of an older version (bbolt) is not moved. See [errors.md](errors.md)

## `[retry.file_data_delete]`, `[retry.object_storage_write]`

Retries per task kind. The wait uses exponential backoff (with ±20% jitter). When either `max_attempts` or `max_duration` is reached, the task ends as Failed. Failed tasks are queued again at every `failed_task_retry_interval`.

| Item | Default for `file_data_delete` | Default for `object_storage_write` | Description |
| --- | --- | --- | --- |
| `base` | `"1s"` | `"1s"` | Wait before the first retry |
| `cap` | `"1m"` | `"10m"` | Upper limit of the wait |
| `max_attempts` | `10` | `20` | Upper limit of attempts |
| `max_duration` | `"1h"` | `"6h"` | Time from queuing the task until giving up |

## `[mount]`

Settings for the mount command.

| Item | Default | Description |
| --- | --- | --- |
| `point` | `""` | Mount point. Can also be specified with `mount -v`. If neither is given, mount stops with an error |
| `debug` | `false` | Debug output from go-fuse. Writes each FUSE request and response to standard error |
| `allow_other` | `false` | Allows users other than the mounting user (including root) to access the mount. The kernel checks permissions by the file mode (`default_permissions`). When mounting as a user other than root, `user_allow_other` is required in `/etc/fuse.conf` |

## `[hrana]`

Settings for the hrana command (SQL front).

| Item | Default | Description |
| --- | --- | --- |
| `listen` | `"127.0.0.1:8080"` | Address on which Hrana over HTTP is served. hrana has no authentication, so localhost is recommended. Other addresses are accepted without restriction |
| `file_data_size_limit` | `1048576` | Largest file data INSERT accepts and SELECT returns (bytes, after base64 decoding). At most the gRPC maximum message size (4 MiB) |
| `file_mode` | `0o644` | Mode of the files INSERT creates |

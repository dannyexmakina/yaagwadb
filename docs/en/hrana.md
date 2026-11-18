# SQL front (hrana)

`yaagwadb hrana` serves Hrana 3 over HTTP (the protocol of libSQL and sqld) and passes it to the daemon over gRPC.
To a libSQL client it looks like SQLite with the table `objects`.

## Start

- Listens on `listen` in `[hrana]` of the configuration file (default `127.0.0.1:8080`)
- No authentication. A `listen` address other than localhost is not refused, so do not use an address reachable from outside
- The daemon must be running
- `query_row_limit` is taken from `[daemon]` in the configuration file hrana reads. Use the same configuration file as the daemon

## Endpoints

| Method and path | Content |
| --- | --- |
| `GET /v3`, `GET /v3-protobuf` | Returns 200 |
| `POST /v3/pipeline` | Handles a list of requests in order (JSON) |
| `POST /v3-protobuf/pipeline` | The same (Protobuf) |
| `POST /v3/cursor` | Runs a batch and returns the results as JSON, one entry per line |
| `POST /v3-protobuf/cursor` | The same (Protobuf, each structure prefixed with its length) |

- Versions 1 and 2, and WebSocket, are not supported
- No stream state is kept. The baton is the empty string, and null in the response to a pipeline that contains `close`
- The cursor runs the whole batch before it returns the results
- A request body that cannot be read is answered with 400

## Supported requests

| Request | Handling |
| --- | --- |
| `execute` | Runs one statement. With `want_rows` false, no rows are returned |
| `batch` | Runs the steps in order. Supports the conditions `ok`, `error`, `not`, `and`, `or`, and `is_autocommit`. A failed step does not stop the batch |
| `describe` | Returns the parameters and the result columns of a statement without running it |
| `get_autocommit` | Always true |
| `close` | Returns success |
| `sequence`, `store_sql`, `close_sql` | Error (`ENOTSUP`) |

- A statement given by `sql_id` is an error
- In a statement result, `rows_read`, `rows_written`, and `query_duration_ms` are always 0, and `last_insert_rowid` is always null
- `describe` gives column names and `decltype` by the same rules as SQLite: an alias if there is one; for a reference to a column, that column's name and declared type (through common table expressions and subqueries); for any other expression, its text as written and no declared type. Of the columns of objects, `size` and `mtime` have the declared type `INTEGER`, and the others have none

## Transactions

- `BEGIN`, `COMMIT`, `ROLLBACK`, and the like are errors (`ENOTSUP`), because the daemon cannot undo several operations together
- A client feature that wraps statements in `BEGIN` and `COMMIT` fails at the `BEGIN`

## Table objects

| Column | Type | Write | Content |
| --- | --- | --- | --- |
| `path` | TEXT | On INSERT only | Path from the root (`docs/a.txt`) |
| `body` | TEXT (base64) | On INSERT only | File data. Returned only when named as a column |
| `size` | INTEGER | No | Size of the file data (bytes) |
| `mtime` | INTEGER | No | Modification time (UNIX nanoseconds) |
| `deleted` | INTEGER | No | Logical deletion flag. Always `0` |
| `xattr` | TEXT | UPDATE | All xattrs as one JSON object. A value outside UTF-8 is `{"base64": "..."}` |

## Statements

| Statement | Content |
| --- | --- |
| `SELECT` | Runs as read SQL of the daemon |
| `INSERT INTO objects (path, body) VALUES (...)` | Writes a file. An error if the path exists |
| `INSERT ... ON CONFLICT (path) DO UPDATE ...` | Overwrites the file |
| `INSERT ... ON CONFLICT DO NOTHING` | Writes nothing if the path exists |
| `UPDATE objects SET xattr = ... WHERE ...` | Replaces all xattrs |
| `DELETE FROM objects WHERE ...` | Logical deletion |
| `BEGIN`, `COMMIT`, `ROLLBACK` | Error |

- Examples
  - `SELECT path, size FROM objects WHERE xattr ->> 'user.tag' = ?`
  - `INSERT INTO objects (path, body) VALUES ('docs/a.txt', 'aGVsbG8=')`
  - `UPDATE objects SET xattr = '{"user.tag":"x"}' WHERE path = 'docs/a.txt'`
- An INSERT into a directory that does not exist is an error
- Files created by INSERT get the mode from `file_mode` in `[hrana]` and the uid and gid of the hrana process (the `yaagwadb` user under systemd)
- UPDATE and DELETE target at most `query_row_limit` rows. More is an error
- BLOB values cannot be used. Pass file data as a base64 string

## Errors

The `code` of an error is the name of an errno.

| code | Content |
| --- | --- |
| `EFBIG` | The file data is over `file_data_size_limit` (INSERT, SELECT) |
| `EINVAL` | Invalid statement, value, or condition. Includes BLOB values, `body` in WHERE or an expression, a LIMIT over `query_row_limit`, and references to tables other than objects |
| `ENOTSUP` | Unsupported request or statement. Includes transactions |
| `ENOENT` | The file or directory does not exist |
| `EEXIST` | A file with the same path exists |
| `EISDIR` | The target is a directory |
| `ENOTDIR` | Part of the path is not a directory |
| `ENOTEMPTY` | The directory is not empty |
| `ENOSPC` | Disk full |
| `EBUSY`, `EIO` | Answer of the daemon (`EIO`: the file data cannot be retrieved, or the daemon is shutting down) |

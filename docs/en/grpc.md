# gRPC

The daemon is operated through gRPC. The mount and hrana use this gRPC too.
The definition is [protocol/yaagwadb.proto](../../protocol/yaagwadb.proto) (service `yaagwadb.YaagwaDB`).

## Connecting

- The Unix domain socket set by `socket` in the configuration file (default `/run/yaagwadb/daemon.sock`)
- No TLS
- The socket has mode `0600`. Connect as the same user as the daemon, or as root (under systemd, hrana runs as the same `yaagwadb` user as the daemon, and the mount runs as root)
- mode, uid, and gid are only recorded and are not used for access control. A client that can connect to the socket can operate on every file
- No server reflection. Tools such as grpcurl need the proto file

## Specifying files

- Files are specified by inode number, not by path
- The root directory is inode `1`
- `Lookup` gives the inode for a parent inode and a name (for `/a/b.txt`: `Lookup(1, "a")`, then `Lookup(<inode of a>, "b.txt")`)
- Names cannot contain `/` or NUL. `.`, `..`, the empty name, and names longer than 255 bytes are not allowed either

## Methods

| Method | Description |
| --- | --- |
| `Lookup` | Gets the attributes for a parent inode and a name |
| `Getattr` | Gets the attributes |
| `Setattr` | Changes mode, uid, gid, atime, and mtime. size accepts only 0 (truncation) |
| `Readdir` | Gets the entries of a directory (stream) |
| `Read` | Reads the file data (stream) |
| `Write` | Writes a file (client stream) |
| `Unlink` | Logically deletes a file |
| `Mkdir`, `Rmdir` | Creates or removes a directory. A directory that is not empty cannot be removed |
| `Rename` | Moves or renames. An entry of the same kind at the destination is replaced (a directory only if it is empty) |
| `Getxattr`, `Setxattr`, `Listxattr`, `Removexattr` | xattr operations. Values are up to 64 KiB |
| `Statfs` | Capacity of the file system holding the data directory |
| `Query` | Runs read SQL on the objects view |
| `Status` | Same as `yaagwadb status` |
| `ListTasks` | Same as `yaagwadb tasks` |

### Write

- Send `target` (parent inode, name, mode, uid, gid) in the first message and `data` (the file data) in the following messages
- Only whole files are written. Partial writes and appends are not supported
- A file with the same name is overwritten. The inode number stays the same, and the old file data is logically deleted
- A directory with the same name is an error
- When the response arrives, the write is complete and `Lookup` finds the file
- The file type in mode is ignored; the file becomes a regular file

### Read

- `offset` and `size` specify the range. `size` 0 means to the end
- Each message is at most 1 MiB

### Unlink

- When the response arrives, `Lookup`, `Read`, and `Readdir` no longer see the file
- The file data is removed after `logical_delete_grace_period`
- A logical deletion cannot be undone

### Query

- One SELECT statement only. The objects view is the only table it can refer to
- Parameters are `parameters` (in the order of `?`) and `named_parameters` (names with their prefix, such as `:name`)
- Values are integer, real, text, or NULL. BLOB is not supported
- At most `query_row_limit` rows are returned, and the statement runs for at most `query_timeout`
- A CTE named `objects` or `yaagwadb_objects_tree` is not allowed

| Column | Type | Content |
| --- | --- | --- |
| `path` | TEXT | Path from the root (`docs/a.txt`). Directories are not included |
| `size` | INTEGER | Size of the file data (bytes) |
| `mtime` | INTEGER | Modification time (UNIX nanoseconds) |
| `deleted` | INTEGER | Logical deletion flag. Always `0`, because logically deleted files are not rows |
| `xattr` | TEXT | All xattrs as one JSON object. A value outside UTF-8 is `{"base64": "..."}` |

## Errors

Errors are returned as a gRPC status code with a `google.rpc.ErrorInfo` detail (domain `yaagwadb`).
Tell errors apart by the reason.

| reason | Code | Description |
| --- | --- | --- |
| `yaagwadb: not found` | `NOT_FOUND` | Does not exist, or logically deleted |
| `yaagwadb: corrupted` | `DATA_LOSS` | The file data cannot be retrieved |
| `yaagwadb: already exists` | `ALREADY_EXISTS` | An entry with the same name exists |
| `yaagwadb: not a directory` | `FAILED_PRECONDITION` | Not a directory |
| `yaagwadb: is a directory` | `FAILED_PRECONDITION` | Is a directory |
| `yaagwadb: directory not empty` | `FAILED_PRECONDITION` | The directory is not empty |
| `yaagwadb: not supported` | `UNIMPLEMENTED` | Unsupported operation (such as truncating to a size other than 0) |
| `yaagwadb: invalid argument` | `INVALID_ARGUMENT` | Invalid name, value, or read SQL |
| `yaagwadb: no space left on device` | `RESOURCE_EXHAUSTED` | Disk full |
| `yaagwadb: daemon closed` | `UNAVAILABLE` | The daemon is shutting down |

`INTERNAL` without a detail is an error inside the daemon. See the daemon log for details.

## Example with grpcurl

Run in the repository root.

```sh
ADDRESS=unix:///run/yaagwadb/daemon.sock
OPTIONS="-plaintext -proto protocol/yaagwadb.proto"

# write "hello\n" to hello.txt (mode 420 is 0o644)
grpcurl $OPTIONS -d @ $ADDRESS yaagwadb.YaagwaDB/Write <<'JSON'
{"target": {"parentInode": 1, "name": "hello.txt", "mode": 420}}
{"data": "aGVsbG8K"}
JSON

grpcurl $OPTIONS -d '{"inode": 1}' $ADDRESS yaagwadb.YaagwaDB/Readdir                          # list the root
grpcurl $OPTIONS -d '{"parentInode": 1, "name": "hello.txt"}' $ADDRESS yaagwadb.YaagwaDB/Lookup  # get the inode by name
grpcurl $OPTIONS -d '{"inode": 2}' $ADDRESS yaagwadb.YaagwaDB/Read                             # read
grpcurl $OPTIONS -d '{"sql": "SELECT path, size FROM objects"}' $ADDRESS yaagwadb.YaagwaDB/Query # read SQL
```

- The address is `unix://` followed by the socket path
- In JSON, bytes are base64 and uint64 values are strings
- The ErrorInfo detail is shown as base64. The reason is also at the start of the message (for example `yaagwadb: not found: "nope" in inode 1`)

## Example in Go

Inside this repository, the `Client` returned by `yaagwadb.Dial` can be used. Tell errors apart with `yaagwadb.Has`.

```go
client, err := yaagwadb.Dial("/run/yaagwadb/daemon.sock")
if err != nil {
	return err
}
defer client.Close()
ctx := context.Background()

_, err = client.Write(ctx,
	&protocol.WriteTarget{ParentInode: yaagwadb.RootInodeNumber, Name: "hello.txt", Mode: 0o644},
	strings.NewReader("hello\n"))
if err != nil {
	return err
}

inode, err := client.Lookup(ctx, yaagwadb.RootInodeNumber, "hello.txt")
if yaagwadb.Has(err, yaagwadb.NotFound) {
	// does not exist
}
data, err := client.Read(ctx, inode.InodeNumber, 0, 0) // to the end
```

- `Client.Read` reads the whole file into memory. For large files, use `client.Service().Read`
- The module name in go.mod is `yaagwadb`, so other projects cannot `go get` it. Other projects and other languages generate a client from the proto file

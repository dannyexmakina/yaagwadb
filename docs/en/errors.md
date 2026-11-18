# Errors

## Configuration file (all commands)

### yaagwadb: config file not found: <path>

The configuration file does not exist.

- Write one with `yaagwadb install`
- If it is not in the default location, specify it with `-c`

### yaagwadb: config file is a directory: <path>

`-c` points to a directory. Specify the path of the configuration file.

### yaagwadb: read config <path>: …

The configuration file cannot be read, or it is not valid TOML.

- If permissions prevent reading, make it readable by the user running the command (under systemd, see "Prerequisites" in [systemd.md](systemd.md))
- Fix syntax errors and type errors (such as a duration written as a number)

## daemon, restore, configtest

### yaagwadb: invalid argument: LogicalDeleteGracePeriod (…) must be longer than IndexArchiveInterval (…) when object storage is configured

Object storage is configured, and `logical_delete_grace_period` is not longer than `index_archive_interval`.

- Make `logical_delete_grace_period` longer than `index_archive_interval`

### yaagwadb: invalid argument: DataDirectory is required

`data_directory` is set to the empty string.

- Set `data_directory`. If omitted, the default is used

### yaagwadb: unknown object storage kind: "…"

`kind` in `[object_storage]` is none of `"s3"`, `"gcs"`, and `""`.

- Fix `kind`

## daemon, restore

### yaagwadb: not supported: the data directory has index.db of an older version (bbolt), which this version does not read

The data directory has index.db of an older version (bbolt). configtest does not detect this.

- There is no way to move the old index.db to the new format
- `restore --force` does not move index.db, so it does not clear this error
- Start with a new data directory

### yaagwadb: data directory is locked by another process: …

Another process is using the same data directory.

- This happens when another daemon or `restore --force` is run while the daemon is running
- Stop the running process first (under systemd, `sudo systemctl stop yaagwadb.service`)

## restore

### yaagwadb: restore moves the local data aside; pass --force to proceed

restore was run without `--force`.

- Read "How restoring works" in [config.md](config.md), then run it with `--force`

### yaagwadb: invalid argument: object storage is not configured

restore was run without object storage configured.

- Configure `[object_storage]`. Without object storage, nothing can be restored

## hrana, configtest

### yaagwadb: hrana file_data_size_limit is over the gRPC message size (… > 4194304 bytes)

`file_data_size_limit` in `[hrana]` is over the gRPC maximum message size.

- Set `file_data_size_limit` to 4194304 or less

### yaagwadb: hrana file_data_size_limit must be positive (…)

`file_data_size_limit` in `[hrana]` is 0 or less.

- Set `file_data_size_limit` to a positive value

## mount, hrana, import

### yaagwadb: cannot connect to the daemon (socket: <path>). …

The daemon cannot be reached.

- Check that the daemon is running
- Check that `socket` in the configuration file is the same as the daemon's. A relative path is resolved against the directory where the command was run
- The socket has mode `0600`. Run as the same user as the daemon, or as root
- If started together with the daemon under systemd, this can happen before the daemon listens. The unit restarts automatically (see [systemd.md](systemd.md))

## import

### yaagwadb: no import source specified. Pass it with -s

No directory to import is specified.

- Specify it with the command-line option `-s`
  - `yaagwadb import -s /path/to/directory`

### yaagwadb: import source is not a directory: <path>

`-s` points to something other than a directory.

- Specify the directory to import. To import a single file, specify a directory that holds only that file

### yaagwadb: import failed: <count> entries

Some entries could not be imported. Each failed entry is shown after `failed:`, with its path relative to the imported directory and the cause.

- `already exists` means the database has an entry of the other kind (file or directory) with the same name. `-f` does not replace it, so clean up the database and run the import again
- For other causes, fix the cause and run the import again. Files already imported are skipped

## mount

### no FUSE mount utility found

(macOS, FreeBSD) FUSE is not installed.

- macOS
  - Install [macFUSE](https://macfuse.github.io/)
- FreeBSD
  - Load the fusefs kernel module
  - `sudo kldload fusefs`

### exec: "/bin/fusermount": stat /bin/fusermount: no such file or directory

(Linux) FUSE is not installed.

- Debian, Ubuntu
  - Install fuse3 (`sudo apt install fuse3`)
- RHEL, Fedora
  - Install fuse3 (`sudo dnf install fuse3`)

### yaagwadb: no mount point specified. Pass it with -v, or set point under [mount] in the configuration file

No mount point is specified, either with `-v` or in the configuration file.

- Specify it with the command-line option `-v`
  - `yaagwadb mount -v /path/to/mountpoint`
- Specify it under `[mount]` in the configuration file
  - `point = "/path/to/mountpoint"`
- If both are given, the command-line option takes precedence

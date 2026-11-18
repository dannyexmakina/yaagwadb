# エラー

## 設定ファイル（全コマンド）

### yaagwadb: config file not found: <パス>

設定ファイルが無い。

- `yaagwadb install` で設定ファイルを書き出す
- 既定以外の場所にある場合は `-c` で指定する

### yaagwadb: config file is a directory: <パス>

`-c` にディレクトリを指定している。設定ファイルのパスを指定する。

### yaagwadb: read config <パス>: …

設定ファイルを読めない、または TOML として解釈できない。

- 権限で読めない場合は、実行ユーザーが読めるようにする（systemd で動かす場合は [systemd.md](systemd.md) の「前提」を参照）
- 構文の誤りや型の誤り（時間を数値で書いたなど）を直す

## daemon・restore・configtest

### yaagwadb: invalid argument: LogicalDeleteGracePeriod (…) must be longer than IndexArchiveInterval (…) when object storage is configured

オブジェクトストレージを使う設定で、`logical_delete_grace_period` が `index_archive_interval` 以下になっている。

- `logical_delete_grace_period` を `index_archive_interval` より長くする

### yaagwadb: invalid argument: DataDirectory is required

`data_directory` が空文字で指定されている。

- `data_directory` を指定する。省略すると既定値になる

### yaagwadb: unknown object storage kind: "…"

`[object_storage]` の `kind` が `"s3"`・`"gcs"`・`""` のどれでもない。

- `kind` を直す

## daemon・restore

### yaagwadb: not supported: the data directory has index.db of an older version (bbolt), which this version does not read

データディレクトリに旧版（bbolt）の index.db がある。configtest では検出されない。

- 旧版の index.db を新しい形式へ移行する手段は無い
- `restore --force` は index.db を移さないため、このエラーは解消しない
- 新しいデータディレクトリで起動する

### yaagwadb: data directory is locked by another process: …

同じデータディレクトリを別のプロセスが使っている。

- daemon が動いている間に、別の daemon や `restore --force` を実行した場合に起きる
- 先に動いているプロセスを停止する（systemd の場合は `sudo systemctl stop yaagwadb.service`）

## restore

### yaagwadb: restore moves the local data aside; pass --force to proceed

`--force` を付けずに実行した。

- 内容を [config.md](config.md) の「リストアの動作」で確認し、`--force` を付けて実行する

### yaagwadb: invalid argument: object storage is not configured

オブジェクトストレージを設定せずに restore を実行した。

- `[object_storage]` を設定する。オブジェクトストレージが無い場合、リストアは不可能

## hrana・configtest

### yaagwadb: hrana file_data_size_limit is over the gRPC message size (… > 4194304 bytes)

`[hrana]` の `file_data_size_limit` が gRPC の最大メッセージサイズを超えている。

- `file_data_size_limit` を 4194304 以下にする

### yaagwadb: hrana file_data_size_limit must be positive (…)

`[hrana]` の `file_data_size_limit` が 0 以下。

- `file_data_size_limit` に正の値を指定する

## mount・hrana・import

### yaagwadb: cannot connect to the daemon (socket: <パス>). …

daemon に接続できない。

- daemon が起動しているか確認する
- 設定ファイルの `socket` が daemon と同じか確認する。相対パスはコマンドを実行したディレクトリを基準に解決される
- socket の権限は `0600`。daemon と同じユーザーか root で実行する
- systemd で daemon と同時に起動した場合は、daemon の待ち受け前に起動して出ることがある。自動で再起動する（[systemd.md](systemd.md) 参照）

## import

### yaagwadb: no import source specified. Pass it with -s

取り込むディレクトリが指定されていない。

- 実行オプション `-s` で指定する
  - `yaagwadb import -s /path/to/directory`

### yaagwadb: import source is not a directory: <パス>

`-s` にディレクトリ以外を指定している。

- 取り込むディレクトリを指定する。ファイル 1 件だけを取り込む場合は、そのファイルだけを置いたディレクトリを指定する

### yaagwadb: import failed: <件数> entries

取り込めなかったエントリがある。失敗したエントリは `failed:` に続けて、取り込むディレクトリからの相対パスと原因で表示される。

- `already exists` は、DB 側に同じ名前で種類の違うエントリ（ファイルとディレクトリ）がある。`-f` でも置き換えないので、DB 側を整理してから再実行する
- その他は原因に応じて対処し、再実行する。取り込み済みのファイルは読み飛ばされる

## mount

### no FUSE mount utility found

(macOS・FreeBSD) FUSE がインストールされていない。

- macOS
  - [macFUSE](https://macfuse.github.io/) をインストールする
- FreeBSD
  - fusefs カーネルモジュールを読み込む
  - `sudo kldload fusefs`

### exec: "/bin/fusermount": stat /bin/fusermount: no such file or directory

(Linux) FUSE がインストールされていない。

- Debian・Ubuntu
  - fuse3 をインストールする（`sudo apt install fuse3`）
- RHEL・Fedora
  - fuse3 をインストールする（`sudo dnf install fuse3`）

### yaagwadb: no mount point specified. Pass it with -v, or set point under [mount] in the configuration file

マウントポイントが `-v` でも設定ファイルでも指定されていない。

- 実行オプション `-v` で指定する
  - `yaagwadb mount -v /path/to/mountpoint`
- 設定ファイルの `[mount]` で指定する
  - `point = "/path/to/mountpoint"`
- 両方ある場合は実行オプションが優先される

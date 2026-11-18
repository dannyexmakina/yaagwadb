# YaagwaDB

[English](#english) | [日本語](#日本語)

## English

YaagwaDB is a file database.
It automatically backs up to object storage (S3, S3-compatible storage, or Google Cloud Storage).

File data lives inside the data directory and cannot be executed.
Files can be restored from object storage.

### Install

FUSE must be installed.

```sh
sudo install -m 0755 yaagwadb /usr/local/bin/yaagwadb   # the systemd units run this path
sudo useradd --system --no-create-home --shell /usr/sbin/nologin yaagwadb   # the daemon and hrana run as this user
sudo yaagwadb install            # writes /etc/yaagwadb/config.toml and the systemd units
sudo chown root:yaagwadb /etc/yaagwadb/config.toml   # the file is written with mode 0640; the yaagwadb user must be able to read it
sudo systemctl daemon-reload
sudo systemctl enable --now yaagwadb.service
sudo systemctl enable --now yaagwadb-mount.service
```

Edit `/etc/yaagwadb/config.toml` before starting the services.
For the configuration items, see [docs/en/config.md](docs/en/config.md)

For the systemd setup, see [docs/en/systemd.md](docs/en/systemd.md).

### Start

Each of the following commands specifies the configuration file with `-c <file>`
(default `/etc/yaagwadb/config.toml`).

- `yaagwadb daemon`
  - Runs as the daemon (gRPC server)
- `yaagwadb mount [-v <mount point>]`
  - Mounts with FUSE as a gRPC client of the daemon
- `yaagwadb hrana`
  - Runs as the SQL front (Hrana over HTTP, compatible with libSQL / sqld) as a gRPC client of the daemon

### Commands

- `yaagwadb install [--unit-directory <directory>]`
  - Writes an example configuration file and the systemd units. Existing files are left unchanged
- `yaagwadb configtest`
  - Checks that the configuration file exists and that its values are valid, then exits
- `yaagwadb status`
  - Shows the number of files and the state of the task queues
- `yaagwadb tasks [--queue <queue>] [--all] [--limit <n>]`
  - Shows the tasks in the queues. Without `--all`, only pending and running tasks
  - `--queue` is `file_data_delete_task_queue` or `object_storage_write_task_queue`. Empty means all queues
  - `--limit` is the number of tasks per queue. `0` means the daemon default
- `yaagwadb import -s <directory> [--to <path in the database>] [-f]`
  - Imports the directories and regular files under the directory as a gRPC client of the daemon. The daemon must be running
  - `--to` is the directory to import into. Missing directories are created. If omitted, the root
  - Files that already exist in the database are skipped. With `-f` (`--force`), they are overwritten
  - Mode, uid, gid, modification time, and xattrs are copied. xattrs in the Linux `security.`, `system.`, and `trusted.` namespaces are not copied
  - Symbolic links, devices, sockets, and other entries that are not regular files or directories are ignored with a warning
  - A failed entry does not stop the import. The failures are shown at the end, and the exit code is 1
- `yaagwadb restore --force`
  - Keeps broken local data under another name and restores the index from object storage. Stop the daemon first
  - File data is not downloaded at this point. Each file data is downloaded when it is read

### Documentation

- [docs/en/config.md](docs/en/config.md)
  - Configuration file
- [docs/en/grpc.md](docs/en/grpc.md)
  - Operating through gRPC
- [docs/en/hrana.md](docs/en/hrana.md)
  - Operating through SQL
- [docs/en/systemd.md](docs/en/systemd.md)
  - Running under systemd
- [docs/en/errors.md](docs/en/errors.md)
  - Errors and how to fix them
- [yaagwadb-test](https://github.com/dannyexmakina/yaagwadb-test)
  - Development environment and tests

## 日本語

YaagwaDB はファイルDB。
オブジェクトストレージ（S3・S3 互換ストレージ・Google Cloud Storage）に自動的にバックアップする。

ファイルデータはデータディレクトリの中にあり、実行不可能
オブジェクトストレージからリストアが可能

### インストール

FUSEのインストールを前提とする

```sh
sudo install -m 0755 yaagwadb /usr/local/bin/yaagwadb   # systemd のユニットはこのパスを実行する
sudo useradd --system --no-create-home --shell /usr/sbin/nologin yaagwadb   # daemon と hrana はこのユーザーで動く
sudo yaagwadb install            # /etc/yaagwadb/config.toml と systemd のユニットを書き出す
sudo chown root:yaagwadb /etc/yaagwadb/config.toml   # 権限 0640 で書き出されるため、yaagwadb ユーザーが読めるようにする
sudo systemctl daemon-reload
sudo systemctl enable --now yaagwadb.service
sudo systemctl enable --now yaagwadb-mount.service
```

サービスを起動する前に `/etc/yaagwadb/config.toml` を編集する。
設定の項目は [docs/ja/config.md](docs/ja/config.md)

systemd の内容については [docs/ja/systemd.md](docs/ja/systemd.md) を参照。

### 起動

以下どのコマンドも `-c <ファイル>` で設定ファイルを指定する
（既定は `/etc/yaagwadb/config.toml`）。

- `yaagwadb daemon`
  - daemon（gRPC サーバー）として起動する
- `yaagwadb mount [-v <マウントポイント>]`
  - daemon の gRPC クライアントとして、FUSE でマウントする
- `yaagwadb hrana`
  - daemon の gRPC クライアントとして、SQL フロント（Hrana over HTTP、libSQL・sqld 互換）を起動する

### コマンド

- `yaagwadb install [--unit-directory <ディレクトリ>]`
  - 設定ファイルの例と systemd のユニットを書き出す。既にあるファイルは書き換えない
- `yaagwadb configtest`
  - 設定ファイルの存在と設定値をチェックして終了する
- `yaagwadb status`
  - ファイルの件数とタスクキューの状態を表示する
- `yaagwadb tasks [--queue <キュー>] [--all] [--limit <件数>]`
  - キューにあるタスクを表示する。`--all` が無い場合は待機中・実行中のタスクのみ
  - `--queue` は `file_data_delete_task_queue` か `object_storage_write_task_queue`。空の場合は全キュー
  - `--limit` はキューごとの件数。`0` の場合は daemon の既定値
- `yaagwadb import -s <ディレクトリ> [--to <DB 内のパス>] [-f]`
  - daemon の gRPC クライアントとして、ディレクトリ以下のディレクトリと通常ファイルを取り込む。daemon が起動している前提
  - `--to` は取り込み先のディレクトリ。無ければ作る。省略した場合はルート
  - DB に既にあるファイルは読み飛ばす。`-f`（`--force`）を付けると上書きする
  - mode・uid・gid・更新時刻・xattr を引き継ぐ。Linux の `security.`・`system.`・`trusted.` 名前空間の xattr は引き継がない
  - シンボリックリンク・デバイス・ソケットなど、通常ファイルとディレクトリ以外は警告を出して無視する
  - 失敗したエントリがあっても最後まで取り込む。失敗は最後に表示し、終了コード 1 で終了する
- `yaagwadb restore --force`
  - 壊れたローカルのデータを別名で残し、オブジェクトストレージから index をリストアする。daemon を停止してから実行する
  - この時点ではファイルデータを取得しない。ファイルデータは読み込まれた時点で1件ずつ取得する

### ドキュメント

- [docs/ja/config.md](docs/ja/config.md)
  - 設定ファイル
- [docs/ja/grpc.md](docs/ja/grpc.md)
  - gRPC での操作
- [docs/ja/hrana.md](docs/ja/hrana.md)
  - SQL での操作
- [docs/ja/systemd.md](docs/ja/systemd.md)
  - systemd での運用
- [docs/ja/errors.md](docs/ja/errors.md)
  - エラーと対処
- [yaagwadb-test](https://github.com/dannyexmakina/yaagwadb-test)
  - 開発環境とテスト

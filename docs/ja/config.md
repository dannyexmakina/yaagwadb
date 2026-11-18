# 設定ファイル

yaagwadb の daemon・mount・hrana は、TOML の設定ファイルを読む。

## 設定ファイルの作成

`yaagwadb install` で、設定ファイルの例を `/etc/yaagwadb/config.toml` に書き出す。`-c` で別の場所を指定可能。

- 既にあれば書き換えない
- 権限 `0640` で書き出される。所有者は実行したユーザー（通常は root）
- systemd のユニットは `/etc/yaagwadb/config.toml` を固定で読む。`-c` で別の場所に書き出した場合は [systemd.md](systemd.md) を参照

## 読み込み

- 置き場所は既定で `/etc/yaagwadb/config.toml`。`-c`（`--config`）でファイル指定可能
- 項目を省略した場合、下の表の既定値となる
- 相対パスは、コマンドを実行したディレクトリを基準に解決する
- 時間は、数値と単位の文字列で記述する
  - 単位は `ns`・`us`・`ms`・`s`・`m`・`h`
  - 組み合わせ使用可（`"1h30m"` は1時間30分）
  - 小数点の使用可（`"1.5h"` は1時間30分）
  - 日の単位は無い。7日は `"168h"` と記述する

## チェック

`yaagwadb configtest` で、設定ファイルの存在と設定値をチェックする。

- 問題が無ければ `<設定ファイルのパス>: configuration OK` を表示して終了する
- 問題があればエラーを表示して、終了コード 1 で終了する。エラーの内容は [errors.md](errors.md) を参照
- 各コマンドも起動時にチェックを行う。範囲はコマンドごとに異なる
  - 設定ファイルの存在：設定ファイルを読む全コマンド（`install` は除く。`restore` は `--force` を指定した場合のみ）
  - daemon の設定値（`[object_storage]` の `kind`、`data_directory` が空でないこと、`logical_delete_grace_period` と `index_archive_interval` の関係）：daemon・`restore`
  - `[hrana]` の `file_data_size_limit`：hrana
  - configtest は上記の全てを行う
- 次の値はチェックしない
  - `log_level` の不正な値。エラーにならず `info` として扱う
  - 数値の範囲（負のワーカー数など）
  - `*_interval` と `scheduler_tick` の大小

## トップレベル

| 項目 | 既定値 | 説明 |
| --- | --- | --- |
| `data_directory` | `"/var/lib/yaagwadb"` | index とタスクキュー（`database_file_name`）・ファイルデータ群（`filedata/`）・一時ファイル（`temporary/`）を置くディレクトリ。systemd で動かす場合の制約は [systemd.md](systemd.md) を参照 |
| `database_file_name` | `"yaagwa.db"` | `data_directory` に置く、index とタスクキューの SQLite ファイルの名前 |
| `socket` | `"/run/yaagwadb/daemon.sock"` | daemon が gRPC を listen する Unix domain socket。systemd で動かす場合の制約は [systemd.md](systemd.md) を参照 |
| `log_level` | `"info"` | ログレベル。`debug`・`info`・`warn`・`error` |

## `[object_storage]`

ファイルデータと index のバックアップを置くオブジェクトストレージの設定。

| 項目 | 既定値 | 説明 |
| --- | --- | --- |
| `kind` | `""` | `"s3"`・`"gcs"` のどちらか。空の場合はバックアップしない |
| `bucket` | `""` | バケット名（`s3`・`gcs`） |
| `prefix` | `""` | オブジェクトストレージ上のファイルのパスのプレフィックス |
| `region` | `""` | `s3` のリージョン。空の場合は AWS SDK の既定の解決順に従う |
| `endpoint` | `""` | 下記「エンドポイント」参照 |

### エンドポイント

- `s3` では、S3 互換ストレージのエンドポイント。バケットをパス形式で指定
- `gcs` では、Cloud Storage のエンドポイント（Private Service Connect など）。空の場合は既定のエンドポイント

### 認証

認証情報は設定ファイルに記述しない。各 SDK の既定の解決順で取得する。

- `s3`
  - AWS 上では IAM ロール（EC2 のインスタンスプロファイル、ECS のタスクロール、EKS の IRSA・Pod Identity）
  - AWS 外では環境変数 `AWS_ACCESS_KEY_ID`・`AWS_SECRET_ACCESS_KEY`（S3 互換ストレージを含む）、または `~/.aws/credentials` のプロファイル（`AWS_PROFILE` で選択）
- `gcs`
  - Google Cloud 上ではアタッチしたサービスアカウント（GCE）、Workload Identity（GKE）
  - Google Cloud 外では Workload Identity 連携の構成ファイル、またはサービスアカウントキーのファイル。パスを環境変数 `GOOGLE_APPLICATION_CREDENTIALS` で指定
  - エミュレーター（fake-gcs-server など）には環境変数 `STORAGE_EMULATOR_HOST` にホストとポートを指定。認証は行わない。`endpoint` は空にする

環境変数は daemon・`restore` の実行環境に設定する。

## `[daemon]`

| 項目 | 既定値 | 説明 |
| --- | --- | --- |
| `file_data_delete_workers` | `2` | ファイルデータ削除 worker の並列数 |
| `object_storage_write_workers` | `8` | オブジェクトストレージ書き込み worker 群の並列数 |
| `scheduler_tick` | `"10s"` | 定期処理を実行するか判定する間隔。各 `*_interval` はこれより短くしても効かない |
| `physical_delete_interval` | `"10s"` | 論理削除済みのファイルデータを物理削除する間隔 |
| `index_archive_interval` | `"15m"` | index をオブジェクトストレージにバックアップする間隔 |
| `finished_task_cleanup_interval` | `"1h"` | 終了済みのタスクを削除する間隔 |
| `logical_delete_grace_period` | `"24h"` | 論理削除から物理削除までの期間。オブジェクトストレージを使う場合は `index_archive_interval` より長くする。長くなければ daemon は起動しない |
| `failed_task_retry_interval` | `"1h"` | リトライ上限に達して Failed で確定したタスクを再試行する間隔 |
| `finished_task_retention` | `"24h"` | 成功したタスクを残しておく期間。過ぎたら `finished_task_cleanup_interval` ごとに削除 |
| `worker_idle_wait` | `"1m"` | 実行可能なタスクが無い場合、worker が次に実行されるまでの最長の待ち時間 |
| `worker_error_wait` | `"1s"` | タスクの取り出しに失敗した場合、worker の次の試行までの時間 |
| `running_task_recovery_interval` | `"10m"` | `running` のまま止まったタスクを確認する間隔 |
| `running_task_timeout` | `"1h"` | `running` のタスクを `pending` に戻すまでの時間。どのタスクの所要時間よりも長くする |
| `database_busy_timeout` | `"5s"` | `database_file_name` のファイルのロックを待つ上限 |
| `query_timeout` | `"10s"` | 読み取り SQL 1 文の実行時間の上限 |
| `query_row_limit` | `1000` | 読み取り SQL 1 文で返す行数の上限。hrana の LIMIT の上限も兼ねる（hrana は自身が読む設定ファイルの値を使う。daemon と同じ設定ファイルを使うこと） |

### ローカル全損時に失う範囲

データディレクトリが全損した場合、オブジェクトストレージからリストアしても次の 2 つは戻らない。

- オブジェクトストレージへの PUT が完了していなかったファイルデータ。待機中・実行中のものと、リトライ上限に達して Failed で確定したものの両方を含む。件数は `yaagwadb status` で確認可能
- 最後に index をバックアップした後の index の変更（新規登録・Rename・論理削除・属性や xattr の変更など）。`index_archive_interval` を短くすると、この範囲は狭まる

index のバックアップより後に書き込まれたファイルデータは、リストアした index から参照されないオブジェクトストレージ上のファイルとして残る。

### リストアの動作

- オブジェクトストレージを設定している場合、daemon は起動時に `yaagwa.db` が無ければ、最新の index をオブジェクトストレージから取得してから起動する。データディレクトリが全損した場合は、daemon を起動するだけでリストアされる
- リストア時にはファイルデータを取得しない。ローカルに無いファイルデータは、読み込まれた時点で1件ずつオブジェクトストレージから取得する
- `yaagwadb restore --force` は、ローカルのデータが壊れて残っている場合に使う。`filedata/` と `yaagwa.db`（`-wal`・`-shm` を含む）を `.broken-<UNIX ナノ秒>` を付けた名前に移し、上記と同じ手順で index をリストアして終了する
  - データディレクトリをロックするため、daemon を停止してから実行する
  - PUT が完了していなかったファイルデータは `filedata.broken-<UNIX ナノ秒>/` に残る
  - 旧版（bbolt）の `index.db` は移さない。[errors.md](errors.md) を参照

## `[retry.file_data_delete]`・`[retry.object_storage_write]`

タスク種別ごとのリトライ。待ち時間は指数バックオフ（±20% のジッター付き）。`max_attempts` か `max_duration` のどちらかに達したら、タスクは Failed で確定する。確定したタスクは `failed_task_retry_interval` ごとに積み直される。

| 項目 | `file_data_delete` の既定値 | `object_storage_write` の既定値 | 説明 |
| --- | --- | --- | --- |
| `base` | `"1s"` | `"1s"` | 1回目の再試行までの待ち時間 |
| `cap` | `"1m"` | `"10m"` | 待ち時間の上限 |
| `max_attempts` | `10` | `20` | 試行回数の上限 |
| `max_duration` | `"1h"` | `"6h"` | タスクを積んでから諦めるまでの時間 |

## `[mount]`

mount コマンドの設定。

| 項目 | 既定値 | 説明 |
| --- | --- | --- |
| `point` | `""` | マウントポイント。`mount -v` での指定が可能。どちらも無い場合はエラー停止 |
| `debug` | `false` | go-fuse のデバッグ出力。FUSE の要求と応答を1件ずつ標準エラー出力 |
| `allow_other` | `false` | マウントしたユーザー以外（root を含む）の参照を許可する。権限はファイルの mode でカーネルが判定（`default_permissions`）。root 以外でマウントする場合、`/etc/fuse.conf` に `user_allow_other` が必要 |

## `[hrana]`

hrana コマンド（SQL フロント）の設定。

| 項目 | 既定値 | 説明 |
| --- | --- | --- |
| `listen` | `"127.0.0.1:8080"` | Hrana over HTTP を受けるアドレス。hrana は認証を持たないため localhost を推奨。他のアドレスも指定可能で、制限はかからない |
| `file_data_size_limit` | `1048576` | INSERT で受け取り、SELECT で返すファイルデータの上限（バイト、base64 デコード後）。gRPC の最大メッセージサイズ（4 MiB）以下 |
| `file_mode` | `0o644` | INSERT で作るファイルのモード |

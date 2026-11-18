# SQL フロント（hrana）

`yaagwadb hrana` は、Hrana 3 over HTTP（libSQL・sqld のプロトコル）を受け、daemon の gRPC に渡す。
libSQL のクライアントからは、表 `objects` を持つ SQLite として見える。

## 起動

- 設定ファイルの `[hrana]` の `listen` で待ち受ける（既定は `127.0.0.1:8080`）
- 認証なし。`listen` に localhost 以外のアドレスを指定しても拒否しないため、外部から届くアドレスにしない
- daemon が起動している前提
- `query_row_limit` は、hrana が読む設定ファイルの `[daemon]` の値を使う。daemon と同じ設定ファイルを使う

## エンドポイント

| メソッドとパス | 内容 |
| --- | --- |
| `GET /v3`・`GET /v3-protobuf` | 200 を返す |
| `POST /v3/pipeline` | 要求の列を順に処理する（JSON） |
| `POST /v3-protobuf/pipeline` | 同上（Protobuf） |
| `POST /v3/cursor` | バッチを実行し、結果を 1 行に 1 件の JSON で返す |
| `POST /v3-protobuf/cursor` | 同上（長さを前置した Protobuf） |

- v1・v2 と WebSocket には対応しない
- ストリームの状態は持たない。baton は空文字で、`close` を含む pipeline の応答では null
- cursor はバッチを最後まで実行してから結果を返す
- 要求の本文を読めない場合は 400

## 対応する要求

| 要求 | 扱い |
| --- | --- |
| `execute` | 1 文を実行。`want_rows` が false の場合は行を返さない |
| `batch` | ステップを順に実行。条件（`ok`・`error`・`not`・`and`・`or`・`is_autocommit`）に対応。失敗したステップがあっても続きを実行する |
| `describe` | 文を実行せずに、パラメーターと結果の列を返す |
| `get_autocommit` | 常に true |
| `close` | 成功を返す |
| `sequence`・`store_sql`・`close_sql` | エラー（`ENOTSUP`） |

- `sql_id` で SQL テキストを指定する文はエラー
- 文の結果の `rows_read`・`rows_written`・`query_duration_ms` は常に 0、`last_insert_rowid` は常に null
- `describe` の列名と `decltype` は SQLite と同じ規則で求める。別名があれば別名、列の参照はその列の名前と宣言型（共通表式・副問い合わせをたどる）、それ以外の式は書かれたままの文字列で宣言型なし。objects の列の宣言型は `size`・`mtime` が `INTEGER`、他は null

## トランザクション

- `BEGIN`・`COMMIT`・`ROLLBACK` などはエラー（`ENOTSUP`）。daemon に複数の操作をまとめて取り消す仕組みが無いため
- 文を `BEGIN` と `COMMIT` で囲むクライアントの機能は、`BEGIN` の時点でエラーになる

## 表 objects

| 列 | 型 | 書き込み | 内容 |
| --- | --- | --- | --- |
| `path` | TEXT | INSERT 時のみ | ルートからのパス（`docs/a.txt`） |
| `body` | TEXT（base64） | INSERT 時のみ | ファイルデータ。列名で指定した場合のみ返す |
| `size` | INTEGER | 不可 | ファイルデータのサイズ（バイト） |
| `mtime` | INTEGER | 不可 | 更新時刻（UNIX ナノ秒） |
| `deleted` | INTEGER | 不可 | 論理削除フラグ。常に `0` |
| `xattr` | TEXT | UPDATE 可 | 全 xattr の JSON オブジェクト。UTF-8 外の値は `{"base64": "..."}` |

## 文

| 文 | 内容 |
| --- | --- |
| `SELECT` | daemon の読み取り SQL として実行 |
| `INSERT INTO objects (path, body) VALUES (...)` | ファイルを書き込む。同じ path があればエラー |
| `INSERT ... ON CONFLICT (path) DO UPDATE ...` | ファイルを上書き |
| `INSERT ... ON CONFLICT DO NOTHING` | 同じ path があれば書き込まない |
| `UPDATE objects SET xattr = ... WHERE ...` | xattr を全て置き換える |
| `DELETE FROM objects WHERE ...` | 論理削除 |
| `BEGIN`・`COMMIT`・`ROLLBACK` | エラー |

- 例
  - `SELECT path, size FROM objects WHERE xattr ->> 'user.tag' = ?`
  - `INSERT INTO objects (path, body) VALUES ('docs/a.txt', 'aGVsbG8=')`
  - `UPDATE objects SET xattr = '{"user.tag":"x"}' WHERE path = 'docs/a.txt'`
- INSERT 先のディレクトリが無い場合はエラー
- INSERT で作るファイルの mode は `[hrana]` の `file_mode`、uid・gid は hrana プロセスのもの（systemd では `yaagwadb` ユーザー）
- UPDATE・DELETE の対象は `query_row_limit` 行まで。超えた場合はエラー
- BLOB 型の値は使えない。ファイルデータは base64 の文字列で渡す

## エラー

エラーの `code` は errno の名前。

| code | 内容 |
| --- | --- |
| `EFBIG` | ファイルデータが `file_data_size_limit` を超える（INSERT・SELECT） |
| `EINVAL` | 文・値・条件が不正。BLOB 型の値、`body` を WHERE や式で使用、`query_row_limit` を超える LIMIT、objects 以外の表への参照を含む |
| `ENOTSUP` | 非対応の要求・文。トランザクションを含む |
| `ENOENT` | 対象のファイル・ディレクトリが無い |
| `EEXIST` | 同じ path のファイルがある |
| `EISDIR` | 対象がディレクトリである |
| `ENOTDIR` | path の途中がディレクトリではない |
| `ENOTEMPTY` | ディレクトリが空でない |
| `ENOSPC` | ディスクフル |
| `EBUSY`・`EIO` | daemon の応答（`EIO` はファイルデータを取得不可能、または daemon が終了処理中） |

# gRPC

daemon は gRPC で操作可能。mount と hrana もこの gRPC を使用する。
定義は [protocol/yaagwadb.proto](../../protocol/yaagwadb.proto)（サービス `yaagwadb.YaagwaDB`）。

## 接続

- 設定ファイルの `socket` の Unix domain socket（既定は `/run/yaagwadb/daemon.sock`）
- TLS なし
- socket の権限は `0600`。daemon と同じユーザーか root で接続する（systemd では hrana は daemon と同じ `yaagwadb` ユーザー、mount は root で動く）
- mode・uid・gid は記録のみで、アクセス制御には使用しない。socket に接続可能なら全てのファイルを操作可能
- サーバーリフレクションなし。grpcurl などでは proto ファイルを指定する

## ファイルの指定

- パスではなく inode 番号で指定する
- ルートディレクトリの inode は `1`
- `Lookup` で親 inode と名前から inode を得る（`/a/b.txt` は `Lookup(1, "a")`、`Lookup(<a の inode>, "b.txt")` の順）
- 名前に `/`・NUL は使用不可。`.`・`..`・空文字・255 バイトを超える名前も不可

## メソッド

| メソッド | 内容 |
| --- | --- |
| `Lookup` | 親 inode と名前から属性を取得 |
| `Getattr` | 属性を取得 |
| `Setattr` | mode・uid・gid・atime・mtime を変更。size は 0（切り詰め）のみ |
| `Readdir` | ディレクトリのエントリを取得（ストリーム） |
| `Read` | ファイルデータを読み込む（ストリーム） |
| `Write` | ファイルを書き込む（クライアントストリーム） |
| `Unlink` | ファイルを論理削除 |
| `Mkdir`・`Rmdir` | ディレクトリの作成・削除。空でないディレクトリは削除不可 |
| `Rename` | 移動・名前の変更。移動先に同じ種類のエントリがあれば置き換える（ディレクトリは空の場合のみ） |
| `Getxattr`・`Setxattr`・`Listxattr`・`Removexattr` | xattr の操作。値は 64 KiB まで |
| `Statfs` | データディレクトリのあるファイルシステムの容量 |
| `Query` | 読み取り SQL を objects ビューに実行 |
| `Status` | `yaagwadb status` と同じ |
| `ListTasks` | `yaagwadb tasks` と同じ |

### Write

- 1 通目に `target`（親 inode・名前・mode・uid・gid）、2 通目以降に `data`（ファイルデータ）を送る
- ファイル全体の書き込みのみ。部分的な書き込み・追記は不可
- 同名のファイルがあれば上書き。inode 番号は変わらず、古いファイルデータは論理削除される
- 同名のディレクトリがあればエラー
- 応答の時点で書き込みは完了しており、`Lookup` で取得可能
- mode のファイル種別は無視され、通常ファイルとなる

### Read

- `offset` と `size` で範囲を指定。`size` が 0 の場合は末尾まで
- 1 メッセージは最大 1 MiB

### Unlink

- 応答の時点で `Lookup`・`Read`・`Readdir` の対象から外れる
- ファイルデータの削除は `logical_delete_grace_period` 経過後
- 論理削除の取り消しは不可

### Query

- SELECT 1 文のみ。参照可能な表は objects ビューのみ
- パラメータは `parameters`（`?` の順）と `named_parameters`（`:name` など、接頭辞を含む名前）
- 値は integer・real・text・NULL。BLOB は非対応
- 返す行数は `query_row_limit` まで。実行時間は `query_timeout` まで
- `objects`・`yaagwadb_objects_tree` という名前の CTE は不可

| 列 | 型 | 内容 |
| --- | --- | --- |
| `path` | TEXT | ルートからのパス（`docs/a.txt`）。ディレクトリは含まない |
| `size` | INTEGER | ファイルデータのサイズ（バイト） |
| `mtime` | INTEGER | 更新時刻（UNIX ナノ秒） |
| `deleted` | INTEGER | 論理削除フラグ。論理削除済みのファイルは行に含まれないため常に `0` |
| `xattr` | TEXT | 全 xattr の JSON オブジェクト。UTF-8 外の値は `{"base64": "..."}` |

## エラー

gRPC のステータスコードと、detail の `google.rpc.ErrorInfo`（domain は `yaagwadb`）で返す。
エラーの種類は reason で判別する。

| reason | コード | 内容 |
| --- | --- | --- |
| `yaagwadb: not found` | `NOT_FOUND` | 存在しない、または論理削除済み |
| `yaagwadb: corrupted` | `DATA_LOSS` | ファイルデータを取得不可能 |
| `yaagwadb: already exists` | `ALREADY_EXISTS` | 同名のエントリがある |
| `yaagwadb: not a directory` | `FAILED_PRECONDITION` | ディレクトリではない |
| `yaagwadb: is a directory` | `FAILED_PRECONDITION` | ディレクトリである |
| `yaagwadb: directory not empty` | `FAILED_PRECONDITION` | ディレクトリが空でない |
| `yaagwadb: not supported` | `UNIMPLEMENTED` | 非対応の操作（0 以外への切り詰めなど） |
| `yaagwadb: invalid argument` | `INVALID_ARGUMENT` | 名前・値・読み取り SQL が不正 |
| `yaagwadb: no space left on device` | `RESOURCE_EXHAUSTED` | ディスクフル |
| `yaagwadb: daemon closed` | `UNAVAILABLE` | daemon が終了処理中 |

detail の無い `INTERNAL` は daemon 内部のエラー。詳細は daemon のログを参照。

## grpcurl での実行例

リポジトリのルートで実行する。

```sh
ADDRESS=unix:///run/yaagwadb/daemon.sock
OPTIONS="-plaintext -proto protocol/yaagwadb.proto"

# 書き込み（"hello\n" を hello.txt に。mode 420 は 0o644）
grpcurl $OPTIONS -d @ $ADDRESS yaagwadb.YaagwaDB/Write <<'JSON'
{"target": {"parentInode": 1, "name": "hello.txt", "mode": 420}}
{"data": "aGVsbG8K"}
JSON

grpcurl $OPTIONS -d '{"inode": 1}' $ADDRESS yaagwadb.YaagwaDB/Readdir                          # ルートの一覧
grpcurl $OPTIONS -d '{"parentInode": 1, "name": "hello.txt"}' $ADDRESS yaagwadb.YaagwaDB/Lookup  # 名前から inode を取得
grpcurl $OPTIONS -d '{"inode": 2}' $ADDRESS yaagwadb.YaagwaDB/Read                             # 読み込み
grpcurl $OPTIONS -d '{"sql": "SELECT path, size FROM objects"}' $ADDRESS yaagwadb.YaagwaDB/Query # 読み取り SQL
```

- アドレスは `unix://` に続けて socket のパスを指定
- JSON では、bytes は base64、uint64 は文字列
- ErrorInfo の detail は base64 のまま表示される。reason はメッセージの先頭にも含まれる（例：`yaagwadb: not found: "nope" in inode 1`）

## Go での実行例

このリポジトリ内では `yaagwadb.Dial` が返す `Client` を使用可能。エラーは `yaagwadb.Has` で判別する。

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
	// 存在しない
}
data, err := client.Read(ctx, inode.InodeNumber, 0, 0) // 末尾まで
```

- `Client.Read` はファイル全体をメモリに読み込む。大きなファイルは `client.Service().Read` を使用
- go.mod のモジュール名が `yaagwadb` のため、他のプロジェクトからは `go get` 不可能。他のプロジェクト・他の言語からは proto ファイルからクライアントを生成する

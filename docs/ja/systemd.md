# systemd

`yaagwadb install` により生成される。

| ユニット | 生成されるパス | 実行するコマンド | 実行ユーザー |
| --- | --- | --- | --- |
| `yaagwadb.service` | `/etc/systemd/system/yaagwadb.service` | daemon（`yaagwadb daemon`） | `yaagwadb` |
| `yaagwadb-mount.service` | `/etc/systemd/system/yaagwadb-mount.service` | mount（`yaagwadb mount`）。`yaagwadb.service` に依存する | root |
| `yaagwadb-hrana.service` | `/etc/systemd/system/yaagwadb-hrana.service` | hrana（`yaagwadb hrana`）。`yaagwadb.service` に依存する | `yaagwadb` |

設定ファイルの項目は [config.md](config.md) を参照。

## 前提

ユニットは次の値を固定で持つ。`yaagwadb install` はこれらを作らない。

- バイナリのパスは `/usr/local/bin/yaagwadb`
- 設定ファイルのパスは `/etc/yaagwadb/config.toml`。`install -c` で別の場所に設定ファイルを書き出しても、ユニットはこのパスを読む。別の場所を使う場合はユニットの `ExecStart` を書き換える
- daemon と hrana は `yaagwadb` ユーザー・グループで動く。事前にユーザーを作る
- 設定ファイルは root 所有・権限 `0640` で書き出される。`yaagwadb` グループが読めるように所有グループを変える

## インストール

```sh
sudo install -m 0755 yaagwadb /usr/local/bin/yaagwadb
sudo useradd --system --no-create-home --shell /usr/sbin/nologin yaagwadb
sudo yaagwadb install
sudo chown root:yaagwadb /etc/yaagwadb/config.toml
sudo systemctl daemon-reload
sudo systemctl enable --now yaagwadb.service
sudo systemctl enable --now yaagwadb-mount.service   # mount を使用する場合
sudo systemctl enable --now yaagwadb-hrana.service   # hrana を使用する場合
```

`yaagwadb install --unit-directory` で別のパスを指定可能

- `yaagwadb install` は、設定ファイルもユニットも既にあれば書き換えない。書き出したファイルは `wrote`、残したファイルは `kept existing` と表示する
- 新しい版のユニットに入れ替える場合は、既存のユニットを削除してから `yaagwadb install` を実行する

## 書き込み可能なディレクトリ

daemon と hrana のユニットは `ProtectSystem=strict` を指定している。書き込めるのは systemd が用意する次のディレクトリのみ。

- `/var/lib/yaagwadb`（`StateDirectory`、権限 `0700`）
- `/run/yaagwadb`（`RuntimeDirectory`、権限 `0755`）

設定ファイルの `data_directory`・`socket` をこれ以外の場所に変える場合は、ユニットに `ReadWritePaths=` を追加する。

## 起動

- mount・hrana の起動時には daemon が起動して応答可能になっていること
- ユニットは `Type=simple` のため、systemd は daemon が socket で待ち受ける前に mount・hrana を起動することがある。その場合 mount・hrana は `yaagwadb: cannot connect to the daemon` で終了し、`Restart=on-failure` により 2 秒後に再起動する
- mount は root で動くため、`0600` の socket に接続可能（[grpc.md](grpc.md) の「接続」参照）

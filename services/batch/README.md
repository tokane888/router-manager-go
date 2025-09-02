# Router Manager Batch Service

定期的にドメインの名前解決を実行し、nftablesルールを更新するバッチサービスです。

## 概要

このサービスは以下の処理を実行します：

1. データベースに登録されたブロック対象ドメインの名前解決
2. 解決されたIPアドレスの管理
3. nftablesファイアウォールルールの自動更新

## ディレクトリ構成

```
.
├── cmd/batch/          # エントリーポイント
├── internal/           # 内部実装
├── debian/             # Debianパッケージ用ファイル
│   ├── control         # パッケージメタデータ
│   ├── rules           # ビルドルール  
│   ├── *.service       # systemdユニットファイル
│   └── *.default       # デフォルト設定
├── deploy/             # 手動デプロイ用（開発用）
└── Makefile           # ビルドタスク
```

## ビルド

### 通常ビルド

```bash
make build
# または
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o router-manager-batch cmd/batch/main.go
```

### Debianパッケージビルド

```bash
# .debファイルを生成
make deb

# または dpkg-buildpackageを直接使用
dpkg-buildpackage -b -rfakeroot -us -uc --host-arch=arm64
```

## デプロイ

### Debianパッケージによるデプロイ（推奨）

```bash
# パッケージのインストール
sudo apt install ./router-manager-batch_*.deb

# 設定の編集
sudo nano /etc/default/router-manager-batch

# サービスの有効化
sudo systemctl enable router-manager-batch.timer
sudo systemctl start router-manager-batch.timer
```

### 手動デプロイ

```bash
# デプロイスクリプトを実行
sudo ./deploy/deploy.sh
```

## 設定

設定ファイルは `/etc/default/router-manager-batch` に配置されます。

主な設定項目：

- `DB_*`: データベース接続設定
- `NFTABLES_*`: nftables関連設定
- `DNS_RESOLVER_*`: DNS解決設定
- `LOG_*`: ログ設定

## 開発

### テスト実行

```bash
make test
# または
go test -v ./...
```

### 開発環境での実行

```bash
make dev-run
# または
go run cmd/batch/main.go
```

### リンティング

```bash
make lint
# または
golangci-lint run ./...
```

### コードフォーマット

```bash
make fmt
```

## トラブルシューティング

### ログの確認

```bash
# サービスログ
sudo journalctl -u router-manager-batch.service -f

# タイマーの状態
sudo systemctl status router-manager-batch.timer
```

### 権限エラーが発生する場合

```bash
# nftables操作権限の確認
getcap /usr/local/bin/router-manager-batch

# 必要に応じて権限を再設定
sudo setcap 'cap_net_admin,cap_net_raw+ep' /usr/local/bin/router-manager-batch
```

### データベース接続エラー

```bash
# PostgreSQLの状態確認
cd /opt/router-manager
sudo docker-compose ps

# PostgreSQLログの確認
sudo docker-compose logs postgres
```

## systemdユニット

- **router-manager-batch.service**: バッチ処理を実行するサービス
- **router-manager-batch.timer**: 毎時0分に実行するタイマー

### 手動実行

```bash
sudo systemctl start router-manager-batch.service
```

### タイマーの無効化

```bash
sudo systemctl stop router-manager-batch.timer
sudo systemctl disable router-manager-batch.timer
```
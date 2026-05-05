# router-manager-go

## 各service概要

- api
  - 下記のようなAPIを提供
    - dnsmasq設定を編集し、名前解決block対象のドメインを追加
    - nftによってipをblockする対象のドメインをDBに登録
- batch
  - 定期的に実行
  - DBに登録されたドメインの名前解決を複数回行い、ドメインに紐づくipの一覧を取得
    - 30秒間隔で最低2回、最大5回名前解決実行
    - 前回と異なる名前解決結果が出なければ終了
      - ラウンドロビンで名前解決結果が切り替わるドメインがあるため
  - DBにドメインに紐づくipが登録されていない場合
    - DBにドメインに紐づくipを登録
    - 当該ipへのpacketのforwardをblock
  - DBにドメインに紐づくipが登録されていた場合
    - 名前解決結果と登録されているipを比較
      - 名前解決結果にのみ含まれるipがあれば、当該ipへのpacketのforwardをblock
      - DBに登録済みだが名前解決結果に含まれないipがあれば、当該ipへのpacketのforwardのblockを解除
      - blockの永続化は行わない(OS再起動で元に戻る)

## 開発環境構築手順

### 必要な環境変数（ローカル環境）

MCPサーバー（cipher）を使用する場合、以下の環境変数をホスト側で設定してください：

- `ANTHROPIC_API_KEY`: Anthropic Claude API キー
- `OLLAMA_BASE_URL`: Ollama ローカルLLM URL

例（~/.zshrc または ~/.bashrc）：

```bash
export ANTHROPIC_API_KEY="sk-ant-xxx..."
```

※未設定の場合、cipher MCPサーバーは接続に失敗しますが、その他の開発作業には影響しません。

### 手順

- devcontainer起動
- 下記実行でcommit前git hook登録
  - `lefthook install`

## 設計方針

- ディレクトリ構成は[Standard Go Project Layout](https://github.com/golang-standards/project-layout/blob/master/README_ja.md#standard-go-project-layout)に従う
- Go モノレポによる複数サービス管理
- 共通モジュールは `pkg/` ディレクトリに配置
  - replace ディレクティブでローカル参照
- 各サービスは独立した go.mod を持つ
- 設計はクリーンアーキテクチャに従う

## 手動実行例

```bash
# batch実行例
cd services/batch
sudo -E /usr/local/go/bin/go run ./cmd/batch/
```

## サービスデバッグ実行例

- ctrl+shift+dで"RUN AND DEBUG"メニューを開く
- 上のメニューからデバッグ実行したいserviceを選択
- F5押下でデバッグ実行

## local環境向けの各種コマンド例

- 開発用postgresログイン
  - `docker exec -it router-manager-go_devcontainer-postgres-1 psql -U postgres -d router_manager`

## Raspberry Pi OSへのデプロイ

### ディレクトリ構成

各サービスは独立したdebian/ディレクトリを持ち、Debianパッケージとしてビルド可能です：

```
services/
├── batch/
│   ├── debian/              # Debianパッケージ用ファイル
│   │   ├── control          # パッケージメタデータ
│   │   ├── rules            # ビルドルール
│   │   ├── router-manager-batch.service    # systemdサービス
│   │   ├── router-manager-batch.timer      # systemdタイマー
│   │   └── router-manager-batch.default    # デフォルト設定
│   └── deploy/              # 手動デプロイ用スクリプト（開発用）
└── api/                     # 将来的に同様の構成
```

### ファイル配置

本番環境では以下の配置となります：

| 種別            | ファイル                     | 配置先                                             |
| --------------- | ---------------------------- | -------------------------------------------------- |
| バイナリ        | router-manager-batch         | `/usr/local/bin/router-manager-batch`              |
| 設定ファイル    | router-manager-batch         | `/etc/default/router-manager-batch`                |
| systemdサービス | router-manager-batch.service | `/lib/systemd/system/router-manager-batch.service` |
| systemdタイマー | router-manager-batch.timer   | `/lib/systemd/system/router-manager-batch.timer`   |
| Docker Compose  | docker-compose.yml           | `/opt/router-manager/docker-compose.yml`           |

### デプロイ手順

#### 1. 前提条件確認

- Raspberry Pi OS (64-bit推奨)
- Docker及びDocker Composeがインストール済み
- dnsmasqがインストール済み
- nftablesがインストール済み

##### 2 バイナリのビルドとインストール

```bash
cd services/batch
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o router-manager-batch cmd/batch/main.go
sudo cp router-manager-batch /usr/local/bin/
sudo chmod 755 /usr/local/bin/router-manager-batch
```

##### 3 設定ファイルの配置

```bash
# Debianパッケージの設定ファイルをコピー
sudo cp debian/router-manager-batch.default /etc/default/router-manager-batch
sudo chmod 600 /etc/default/router-manager-batch
sudo chown router-manager:router-manager /etc/default/router-manager-batch

# 設定ファイルを編集
sudo nano /etc/default/router-manager-batch
```

##### 4 Docker Composeの配置

```bash
sudo mkdir -p /opt/router-manager
sudo cp docker-compose.production.yml /opt/router-manager/docker-compose.yml
sudo chown -R router-manager:router-manager /opt/router-manager

# PostgreSQLを起動
cd /opt/router-manager
sudo docker-compose up -d
```

##### 5 systemdユニットの配置と有効化

```bash
# ユニットファイルをコピー
sudo cp debian/router-manager-batch.service /lib/systemd/system/
sudo cp debian/router-manager-batch.timer /lib/systemd/system/

# systemdをリロード
sudo systemctl daemon-reload

# タイマーを有効化・開始（毎時0分に実行）
sudo systemctl enable router-manager-batch.timer
sudo systemctl start router-manager-batch.timer
```

### 運用コマンド

```bash
# タイマーの状態確認
sudo systemctl status router-manager-batch.timer

# サービスの状態確認
sudo systemctl status router-manager-batch.service

# 手動実行
sudo systemctl start router-manager-batch.service

# ログの確認
sudo journalctl -u router-manager-batch.service -f

# タイマーの次回実行時刻確認
sudo systemctl list-timers router-manager-batch.timer
```

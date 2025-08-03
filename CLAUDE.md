# CLAUDE.md

このファイルは、ルーター管理システムのリポジトリでコードを扱う際のClaude Code (claude.ai/code) への指針を提供します。

## リポジトリ概要

このプロジェクトは、dnsmasqとnftablesを使用してドメインブロッキング機能を提供するルーター管理システムです。APIサービスでブロック対象の管理を行い、バッチサービスで定期的にIPアドレスの解決とファイアウォールルールの更新を行います。

## プロジェクト構成

```text
.
├── services/           # マイクロサービス
│   ├── api/           # APIサービス（ブロック対象管理）
│   │   ├── cmd/api/   # APIサービスのエントリーポイント
│   │   └── internal/  # API内部実装
│   └── batch/         # バッチサービス（定期実行処理）
│       ├── cmd/batch/ # バッチサービスのエントリーポイント
│       └── internal/  # バッチ内部実装
├── pkg/               # サービス間で共有されるパッケージ
│   └── logger/        # uber/zapを使用した共有ロギングパッケージ
├── .devcontainer/     # VS Code DevContainer設定
└── .github/           # GitHub Actionsワークフロー
```

## サービス概要

### APIサービス

- dnsmasq設定を編集し、名前解決ブロック対象のドメインを追加
- nftablesによってIPをブロックする対象のドメインをDBに登録

### バッチサービス

- 定期的に実行される処理
- DBに登録されたドメインの名前解決を実行
- 解決されたIPアドレスに基づいてnftablesルールを更新
  - 新規IPの場合：DBに登録し、パケットフォワードをブロック
  - 既存IPの場合：名前解決結果と比較し、必要に応じてルールを追加/削除

## 開発コマンド

### サービスの実行

```bash
# APIサービスの実行
cd services/api
go run cmd/api/main.go

# バッチサービスの実行
cd services/batch
go run cmd/batch/main.go
```

### デバッグ実行

1. Ctrl+Shift+D で "RUN AND DEBUG" メニューを開く
2. 上部のドロップダウンからデバッグ実行したいサービスを選択
3. F5 押下でデバッグ実行開始

### リンティング

```bash
# golangci-lintの実行（任意のGoモジュールディレクトリから）
golangci-lint run

# 一部の問題を自動修正
golangci-lint run --fix
```

### フォーマット

```bash
# Goコードのフォーマット（golangci-lintのgofumptで処理）
golangci-lint run --fix

# その他のファイル（JSON、Markdown、YAML、TOML）のフォーマット
dprint fmt

# 変更なしでフォーマットをチェック
dprint check
```

### モジュール管理

```bash
# サービスの依存関係を更新
cd services/api
go mod tidy

# すべてのモジュールを更新
find . -name go.mod -exec dirname {} \; | xargs -I {} sh -c 'cd {} && go mod tidy'
```

### Gitフック

```bash
# gitフックのインストール（リポジトリルートから実行）
lefthook install

# フックを手動で実行
lefthook run pre-commit
lefthook run pre-push
```

## アーキテクチャの決定事項

1. **モノレポ構造**: APIとバッチサービスを単一リポジトリで管理し、共有コードの管理と一貫したツール使用を容易にしています。

2. **クリーンアーキテクチャ**: 各サービスはクリーンアーキテクチャの原則に従い、ビジネスロジックを独立させています。

3. **内部パッケージ**: 各サービスは `internal/` ディレクトリを使用して、他のサービスがプライベート実装の詳細をインポートすることを防ぎます。

4. **モジュール境界**: 各サービスは独自の `go.mod` ファイルを持ち、開発中はローカルパッケージ用の `replace` ディレクティブを使用します。

5. **設定管理**: godotenvを使用して `.env/.env.{ENV}` ファイルから環境固有の設定を読み込みます。

6. **構造化ログ**: すべてのサービスがzapを使用した共有ロガーパッケージを使用し、本番環境で一貫したJSONログを出力します。

## 主要な設定ファイル

- `.golangci.yml`: セキュリティチェック、エラー処理、スタイル適用を含む包括的なリンティングルール
- `dprint.json`: Go以外のファイルのフォーマットルール
- `.lefthook.yml`: 自動フォーマットとリンティング用のGitフック設定
- `.devcontainer/devcontainer.json`: すべてのツールがプリインストールされたVS Code開発環境

## システム要件

- Linux環境（dnsmasq、nftablesが必要）
- Go 1.24以上
- DBアクセス（詳細は各サービスの設定を参照）

## セキュリティ考慮事項

- nftablesルールの変更には適切な権限が必要
- dnsmasq設定の変更には管理者権限が必要
- DBアクセス情報は環境変数で管理

## 開発環境構築

1. DevContainerの起動
2. Git hooksの登録: `lefthook install`
3. 環境変数の設定（`.env`ファイルの作成）
4. 必要な権限の確認（nftables、dnsmasq操作権限）

## ソース編集時の注意点

- 対応するソースが残っている状態で日本語のコメントのみを消去しない
- *.goの編集が一通り完了した後、一度ビルドが成功することを確認する
- nftablesやdnsmasqの操作を含むコードは、テスト環境で動作確認を行う

# important-instruction-reminders

Do what has been asked; nothing more, nothing less.
NEVER create files unless they're absolutely necessary for achieving your goal.
ALWAYS prefer editing an existing file to creating a new one.
NEVER proactively create documentation files (*.md) or README files. Only create documentation files if explicitly requested by the User.

## ソース編集時注意点

- 対応するソース残っている状態で日本語のコメントのみを消去しない

## 動作確認

- 編集対象のプロセスのgo.modがあるディレクトリで`golangci-lint run ./...`を実行し、警告が出ないことを確認
- publicメソッドは非常に単純なものを除いて基本的に単体テスト実装

# CLAUDE.md

このファイルは、このリポジトリでコードを扱う際のClaude Code (claude.ai/code) への指針を提供します。

## リポジトリ概要

これは共有パッケージを持つマイクロサービス構築用のGoモノレポテンプレートです。[Standard Go Project Layout](https://github.com/golang-standards/project-layout) に従い、開発ツールが事前設定されています。

## プロジェクト構成

```text
.
├── services/           # 個別のマイクロサービス
│   └── api/           # APIサービスの例
│       ├── cmd/api/   # アプリケーションエントリーポイント (main.go)
│       └── internal/  # プライベートアプリケーションコード
├── pkg/               # サービス間で共有されるパッケージ
│   └── logger/        # uber/zapを使用した共有ロギングパッケージ
├── .devcontainer/     # VS Code DevContainer設定
└── .github/           # GitHub Actionsワークフロー
```

## 開発コマンド

### サービスの実行

```bash
# サービスディレクトリから実行（例：services/api/）
go run cmd/api/main.go
```

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

1. **モノレポ構造**: サービスとパッケージを単一リポジトリで管理し、共有コードの管理と一貫したツール使用を容易にしています。

2. **内部パッケージ**: 各サービスは `internal/` ディレクトリを使用して、他のサービスがプライベート実装の詳細をインポートすることを防ぎます。

3. **モジュール境界**: 各サービスは独自の `go.mod` ファイルを持ち、開発中はローカルパッケージ用の `replace` ディレクティブを使用します。

4. **設定管理**: godotenvを使用して `.env/.env.{ENV}` ファイルから環境固有の設定を読み込みます。

5. **構造化ログ**: すべてのサービスがzapを使用した共有ロガーパッケージを使用し、本番環境で一貫したJSONログを出力します。

## 主要な設定ファイル

- `.golangci.yml`: セキュリティチェック、エラー処理、スタイル適用を含む包括的なリンティングルール
- `dprint.json`: Go以外のファイルのフォーマットルール
- `.lefthook.yml`: 自動フォーマットとリンティング用のGitフック設定
- `.devcontainer/devcontainer.json`: すべてのツールがプリインストールされたVS Code開発環境

## テストのアプローチ

テンプレートにはテストファイルは含まれていません。テストを追加する際は：

- ユニットテストはコードファイルと同じ場所に配置（例：`config_test.go`）
- テストヘルパーは `internal/testutil/` を使用
- サービスディレクトリから `go test ./...` でテストを実行

## CI/CDパイプライン

GitHub Actionsワークフローが以下を処理します：

- すべてのGoモジュールでgolangci-lintを実行
- dprintでコードフォーマットをチェック
- サービス間でのマトリックスビルド
- 自動PRチェック

## 開発環境

DevContainerが提供するもの：

- Go 1.24
- golangci-lint
- dprint
- lefthook
- Git設定
- Claude CodeとGitHub Copilotのサポート

## ソース編集時注意点

- 対応するソース残っている状態で日本語のコメントのみを消去しない
- *.goの編集が一通り完了した後一度build成功することを確認する

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
- github issueで修正を行い`git commit`する場合、timezoneはJSTを使用

## 動作確認

- 下記を実行して整形
  - `gofumpt -w .`
- 下記を実行し、spell check
  - `cspell .`
- build成功を確認
- `go test`実行
- 編集対象のプロセスのgo.modがあるディレクトリで`golangci-lint run ./...`を実行し、警告が出ないことを確認
- publicメソッドは非常に単純なものを除いて基本的に単体テスト実装

## テスト

- 基本的にテーブル駆動方式で記載
- 単一の関数をテストするテストは`Test_validateConfig()`のように`Test_`の後に関数名を記載する形の関数名にする

# Claude Code Spec-Driven Development

Kiro-style Spec Driven Development implementation using claude code slash commands, hooks and agents.

## Project Context

### Paths

- Steering: `.kiro/steering/`
- Specs: `.kiro/specs/`
- Commands: `.claude/commands/`

### Steering vs Specification

**Steering** (`.kiro/steering/`) - Guide AI with project-wide rules and context\
**Specs** (`.kiro/specs/`) - Formalize development process for individual features

### Active Specifications

- Check `.kiro/specs/` for active specifications
- Use `/kiro:spec-status [feature-name]` to check progress

## Development Guidelines

- Think in English, but generate responses in Japanese (思考は英語、回答の生成は日本語で行うように)

## Workflow

### Phase 0: Steering (Optional)

`/kiro:steering` - Create/update steering documents
`/kiro:steering-custom` - Create custom steering for specialized contexts

**Note**: Optional for new features or small additions. Can proceed directly to spec-init.

### Phase 1: Specification Creation

1. `/kiro:spec-init [detailed description]` - Initialize spec with detailed project description
2. `/kiro:spec-requirements [feature]` - Generate requirements document
3. `/kiro:spec-design [feature]` - Interactive: "requirements.mdをレビューしましたか？ [y/N]"
4. `/kiro:spec-tasks [feature]` - Interactive: Confirms both requirements and design review

### Phase 2: Progress Tracking

`/kiro:spec-status [feature]` - Check current progress and phases

## Development Rules

1. **Consider steering**: Run `/kiro:steering` before major development (optional for new features)
2. **Follow 3-phase approval workflow**: Requirements → Design → Tasks → Implementation
3. **Approval required**: Each phase requires human review (interactive prompt or manual)
4. **No skipping phases**: Design requires approved requirements; Tasks require approved design
5. **Update task status**: Mark tasks as completed when working on them
6. **Keep steering current**: Run `/kiro:steering` after significant changes
7. **Check spec compliance**: Use `/kiro:spec-status` to verify alignment

## Steering Configuration

### Current Steering Files

Managed by `/kiro:steering` command. Updates here reflect command changes.

### Active Steering Files

- `product.md`: Always included - Product context and business objectives
- `tech.md`: Always included - Technology stack and architectural decisions
- `structure.md`: Always included - File organization and code patterns

### Custom Steering Files

<!-- Added by /kiro:steering-custom command -->
<!-- Format:
- `filename.md`: Mode - Pattern(s) - Description
  Mode: Always|Conditional|Manual
  Pattern: File patterns for Conditional mode
-->

### Inclusion Modes

- **Always**: Loaded in every interaction (default)
- **Conditional**: Loaded for specific file patterns (e.g., `"*.test.js"`)
- **Manual**: Reference with `@filename.md` syntax

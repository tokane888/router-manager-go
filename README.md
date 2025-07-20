# router-manager-go

- router管理用の各種ツール

## 開発環境構築手順

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

## サービス実行例

```bash
# API サービスの実行
cd services/api
go run ./cmd/app
```

## サービスデバッグ実行例

- ctrl+shift+dで"RUN AND DEBUG"メニューを開く
- 上のメニューからデバッグ実行したいserviceを選択
- F5押下でデバッグ実行

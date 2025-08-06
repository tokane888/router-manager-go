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

## local環境向けの各種コマンド例

- 開発用postgresログイン
  - `docker exec -it router-manager-go_devcontainer-postgres-1 psql -U postgres -d router_manager`

# DevContainer通知設定ガイド

## 概要

DevContainer内から`notify-send`を使用してホストシステムに通知を送信するための設定方法です。

## 設定手順

### 1. 基本的な通知機能（コンテナ内のみ）

Dockerfileに`libnotify-bin`パッケージをインストールする設定を追加しました。これにより、コンテナ内で`notify-send`コマンドが使用可能になります。

```bash
# テスト方法
notify-send "Test" "This is a test notification"
```

### 2. ホストシステムへの通知転送（オプション）

ホストシステムに通知を送信したい場合は、以下のいずれかの方法を使用できます：

#### 方法A: DBusソケットの共有

`devcontainer.json`に以下のマウント設定を追加：

```json
"mounts": [
  // 既存のマウント設定...
  "source=/run/user/1000/bus,target=/run/user/1000/bus,type=bind",
  "source=${localEnv:HOME}/.Xauthority,target=/home/vscode/.Xauthority,type=bind,consistency=cached"
],
"containerEnv": {
  "DISPLAY": "${localEnv:DISPLAY}",
  "DBUS_SESSION_BUS_ADDRESS": "unix:path=/run/user/1000/bus"
}
```

#### 方法B: SSH転送を使用

1. ホストシステムでSSHサーバーを有効化
2. DevContainerから`ssh`経由で通知を送信：

```bash
ssh -o StrictHostKeyChecking=no host.docker.internal notify-send "Title" "Message"
```

#### 方法C: カスタム通知サービスの使用

`postStartCommand.sh`に以下のようなスクリプトを追加：

```bash
# 通知転送用のエイリアス
echo 'alias notify-send-host="curl -X POST http://host.docker.internal:8888/notify -d"' >> ~/.bashrc
```

## 注意事項

- ホストのバイナリを直接マウントする方法は、アーキテクチャやライブラリの互換性問題により推奨されません
- X11/DBus転送を使用する場合は、セキュリティ上の考慮が必要です
- 最もシンプルな方法は、コンテナ内で`libnotify-bin`をインストールすることです

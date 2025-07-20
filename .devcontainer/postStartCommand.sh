#!/bin/bash

USER=$(whoami)

# hostの~/.ssh​が/tmp/.sshへマウントされているので、それをHOMEディレクトリへコピー。
# 直接HOMEディレクトリにマウントしようとすると既存のものがあり失敗するため
cp -r /tmp/.ssh "${HOME}"
chown -R "${USER}":"${USER}" "${HOME}"/.ssh
chmod 600 "${HOME}"/.ssh/*

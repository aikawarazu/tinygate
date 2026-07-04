#!/bin/bash
set -e
cd "$(dirname "$0")/.."

PASS="${1:?Usage: bash scripts/deploy-to-suqinxia.sh <ssh-password>}"

go build -o tinygate .

sshpass -p "$PASS" scp -P 15022 -o StrictHostKeyChecking=no \
    tinygate config.yaml scripts/tinygate.service root@192.168.31.38:/tmp/

sshpass -p "$PASS" ssh -p 15022 -o StrictHostKeyChecking=no root@192.168.31.38 "
    mv /tmp/tinygate /usr/local/bin/tinygate
    mv /tmp/config.yaml /etc/tinygate
    mv /tmp/tinygate.service /etc/systemd/system/
    chmod +x /usr/local/bin/tinygate
    systemctl daemon-reload
    systemctl enable tinygate
    systemctl restart tinygate
    systemctl status tinygate
"

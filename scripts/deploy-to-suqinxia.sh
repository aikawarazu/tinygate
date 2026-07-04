#!/bin/bash
set -e
cd "$(dirname "$0")/.."

PASS="${1:?Usage: bash scripts/deploy-to-suqinxia.sh <ssh-password>}"
HOST=192.168.31.38
PORT=15022
USER=root
DEPLOY_DIR=/etc/tinygate

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tinygate .

sshpass -p "$PASS" ssh -p $PORT -o StrictHostKeyChecking=no $USER@$HOST "mkdir -p $DEPLOY_DIR"

sshpass -p "$PASS" scp -P $PORT -o StrictHostKeyChecking=no \
    tinygate config.yaml scripts/tinygate.service $USER@$HOST:$DEPLOY_DIR/

sshpass -p "$PASS" ssh -p $PORT -o StrictHostKeyChecking=no $USER@$HOST "
    chmod +x $DEPLOY_DIR/tinygate
    cp $DEPLOY_DIR/tinygate.service /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable tinygate
    systemctl restart tinygate
    echo '=== status ==='
    systemctl status tinygate --no-pager
"
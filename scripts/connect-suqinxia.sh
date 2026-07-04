#!/bin/bash
set -e

export SSH_HOST=192.168.31.38 SSH_PORT=15022 SSH_USER=root
export SSH_KEY=/root/.ssh/id_rsa
export LOCAL_PORT=62222 REMOTE_PORT=22

./fsprovider --debug --http-only &
PID=$!

for i in $(seq 1 30); do
    timeout 1 bash -c "echo >/dev/tcp/localhost/$LOCAL_PORT" 2>/dev/null && break
    sleep 1
done

if kill -0 $PID 2>/dev/null; then
    echo "Ready: localhost:$LOCAL_PORT -> suqinxia:$REMOTE_PORT"
    wait $PID
else
    echo "Failed"
    exit 1
fi

#!/bin/bash
set -e
cd "$(dirname "$0")/.."

[ -f /etc/tinygate ] || cp config.yaml /etc/tinygate
export TINYGATE_API_KEYS=noauth

if [ -n "${SUQINXIA_SSH_PASSWORD:-}" ]; then
    SSH_HOST=192.168.31.38 SSH_PORT=15022 SSH_USER=root \
    SSH_PASSWORD="$SUQINXIA_SSH_PASSWORD" \
    LOCAL_PORT=62222 REMOTE_HOST=localhost REMOTE_PORT=22 \
    ./fsprovider --debug &
    TUNNEL_PID=$!
    sleep 2
fi

./tinygate &
GW_PID=$!
echo "Gateway :39901 (PID $GW_PID)"

trap "kill ${GW_PID:-} ${TUNNEL_PID:-} 2>/dev/null" EXIT
wait

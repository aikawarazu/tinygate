#!/bin/bash
# Start full stack: SSH tunnel + gateway
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

# 1. Copy config to default location
cp config.yaml /etc/tinygate 2>/dev/null || true

# 2. Set noauth for gateway
export TINYGATE_API_KEYS=noauth

# 3. Start SSH tunnel (if suqinxia password is available)
if [ -n "${SUQINXIA_SSH_PASSWORD:-}" ]; then
    echo "Starting SSH tunnel to suqinxia..."
    SSH_HOST=192.168.31.38 SSH_PORT=15022 SSH_USER=root \
    SSH_PASSWORD="$SUQINXIA_SSH_PASSWORD" \
    LOCAL_PORT=62222 REMOTE_HOST=localhost REMOTE_PORT=22 \
    ./fsprovider --debug &
    TUNNEL_PID=$!
    sleep 2
    echo "SSH tunnel PID: $TUNNEL_PID"
else
    echo "SKIP: SUQINXIA_SSH_PASSWORD not set"
fi

# 4. Start gateway
echo "Starting gateway on :39901..."
./tinygate --config config.yaml &
GW_PID=$!
echo "Gateway PID: $GW_PID"

# 5. Wait
trap "kill $GW_PID ${TUNNEL_PID:-} 2>/dev/null" EXIT
wait

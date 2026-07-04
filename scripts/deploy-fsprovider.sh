#!/bin/bash
set -e
cd "$(dirname "$0")/.."

PASS="${1:?Usage: bash scripts/deploy-fsprovider.sh <ssh-password>}"
HOST=192.168.31.38
PORT=15022
USER=root
DEPLOY_DIR=/etc/tinygate

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o fsprovider ./cmd/fsprovider/

python3 -c "
import paramiko, os, sys
host = '$HOST'; port = $PORT; user = '$USER'; pw = '$PASS'
remote_dir = '$DEPLOY_DIR'
local_file = 'fsprovider'
service_file = 'scripts/fsprovider.service'

# 1. Create dir
t = paramiko.Transport((host, port))
t.connect(username=user, password=pw)
sftp = paramiko.SFTPClient.from_transport(t)
try: sftp.stat(remote_dir)
except: sftp.mkdir(remote_dir)

# 2. Upload binary
sftp.put(local_file, f'{remote_dir}/fsprovider')
sftp.chmod(f'{remote_dir}/fsprovider', 0o755)

# 3. Upload service
sftp.put(service_file, f'{remote_dir}/fsprovider.service')
sftp.close()
t.close()

# 4. Setup and start
ssh = paramiko.SSHClient()
ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
ssh.connect(host, port=port, username=user, password=pw)
cmds = [
    f'cp {remote_dir}/fsprovider.service /etc/systemd/system/',
    'systemctl daemon-reload',
    'systemctl enable fsprovider',
    'systemctl restart fsprovider',
    'systemctl status fsprovider --no-pager',
]
for cmd in cmds:
    stdin, stdout, stderr = ssh.exec_command(cmd)
    print(stdout.read().decode())
ssh.close()
print('Deploy OK')
"

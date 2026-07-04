# TinyGate Review Plan

## Final Architecture

### Gateway (`./tinygate`)
- Flag: `--config` (default `/etc/tinygate`)
- Always logs: `→ downstream: URL` + `POST /path STATUS DURATION`
- No auto-config generation — config must exist or fatal error

### fsprovider (`./fsprovider`)
- Flag: `--debug` — all SSH config from env vars
- Env: `SSH_HOST`, `SSH_USER`, `SSH_KEY` (or `SSH_PASSWORD`), `LOCAL_PORT`, `REMOTE_HOST`, `REMOTE_PORT`, `SSH_PORT`

## Steps

1. ✅ Gateway: `--config /etc/tinygate`, always log
2. ✅ fsprovider: `--debug`, env config
3. ✅ CI: GOPROXY fix
4. ✅ go.mod: tidy
5. 🔲 Submit CNB merge request → https://cnb.cool/v0.1/tinygate/-/compare/main...feat/noauth-and-versionless-url

## SSH Tunnel Setup Plan

### Prerequisites
- SSH key: `/root/.ssh/id_rsa` (RSA 3072, verified ✅)
- suqinxia IP (pending)

### Steps
1. `echo "<IP> suqinxia" >> /etc/hosts`
2. Test connectivity: `ssh -i /root/.ssh/id_rsa root@suqinxia hostname`
3. Start tunnel:
   ```
   SSH_HOST=suqinxia SSH_USER=root SSH_KEY=/root/.ssh/id_rsa \
   LOCAL_PORT=62222 REMOTE_PORT=22 \
   ./fsprovider --debug --http-only
   ```
4. Verify: `curl -v telnet://localhost:62222`
5. Test full chain with `scripts/curl-opencode.sh`

#!/bin/bash
# TinyGate Smoke Test
set -e

BASE="http://localhost:39901"
KEY="sk-gateway-key-1"
PASS=0
FAIL=0

green() { echo -e "\033[32m  PASS\033[0m $1"; }
red()   { echo -e "\033[31m  FAIL\033[0m $1"; }

check() {
    local name="$1" code="$2" expect_code="$3" body="$4" expect_body="$5"
    local ok=true

    if [ "$code" != "$expect_code" ]; then
        red "$name — status $code (expected $expect_code)"
        ok=false
    fi

    if [ -n "$expect_body" ] && ! echo "$body" | grep -q "$expect_body"; then
        red "$name — body mismatch (expected '$expect_body')"
        echo "         got: $(echo "$body" | head -c 150)"
        ok=false
    fi

    if $ok; then
        green "$name"
        PASS=$((PASS + 1))
    else
        FAIL=$((FAIL + 1))
    fi
}

do_get() {
    local url="$1" auth="$2"
    if [ "$auth" = "none" ]; then
        curl -s -w "\n%{http_code}" "$url"
    else
        curl -s -w "\n%{http_code}" -H "Authorization: Bearer $auth" "$url"
    fi
}

echo " TinyGate Smoke Test"
echo "══════════════════════════════════════════"
echo ""

# ── Health ──
echo "── Health ──"
resp=$(do_get "$BASE/health" "none")
code=$(echo "$resp" | tail -1); body=$(echo "$resp" | head -n -1)
check "health status + body" "$code" 200 "$body" "OK"

# ── Auth ──
echo "── Auth ──"
resp=$(do_get "$BASE/opencode/zen/go/v1/models" "$KEY")
code=$(echo "$resp" | tail -1); body=$(echo "$resp" | head -n -1)
check "valid key — 200 + JSON data" "$code" 200 "$body" '"data"'

resp=$(do_get "$BASE/opencode/zen/go/v1/models" "sk-wrong")
code=$(echo "$resp" | tail -1); body=$(echo "$resp" | head -n -1)
check "invalid key — 401 + Unauthorized" "$code" 401 "$body" "Unauthorized"

resp=$(do_get "$BASE/opencode/zen/go/v1/models" "none")
code=$(echo "$resp" | tail -1); body=$(echo "$resp" | head -n -1)
check "no key — 401 + Unauthorized" "$code" 401 "$body" "Unauthorized"

# ── Routing ──
echo "── Routing ──"
resp=$(do_get "$BASE/opencode/zen/go/v1/models" "$KEY")
code=$(echo "$resp" | tail -1); body=$(echo "$resp" | head -n -1)
check "opencode — 200 + model list" "$code" 200 "$body" '"data"'

resp=$(do_get "$BASE/zhipu/v4/models" "$KEY")
code=$(echo "$resp" | tail -1); body=$(echo "$resp" | head -n -1)
check "zhipu — 200 (proxied OK)" "$code" 200

resp=$(do_get "$BASE/mimo/v1/models" "$KEY")
code=$(echo "$resp" | tail -1); body=$(echo "$resp" | head -n -1)
check "mimo — 200 (proxied OK)" "$code" 200

resp=$(do_get "$BASE/nowhere" "$KEY")
code=$(echo "$resp" | tail -1); body=$(echo "$resp" | head -n -1)
check "404 — Not Found" "$code" 404 "$body" "Not Found"

echo ""
echo "══════════════════════════════════════════"
echo " $PASS passed, $FAIL failed"
echo "══════════════════════════════════════════"

[ "$FAIL" -eq 0 ] && green "All gateway tests passed!" && exit 0
red "Some tests failed." && exit 1

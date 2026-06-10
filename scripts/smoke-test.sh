#!/bin/bash
# TinyGate Smoke Test — tests gateway itself, not downstream APIs
set -e

BASE="http://localhost:39901"
KEY="sk-gateway-key-1"
PASS=0
FAIL=0

green() { echo -e "\033[32m  PASS\033[0m $1"; }
red()   { echo -e "\033[31m  FAIL\033[0m $1"; }

check() {
    local name="$1" code="$2" expect="$3"
    if [ "$code" = "$expect" ]; then
        green "$name ($code)" && PASS=$((PASS + 1))
    else
        red "$name (expected $expect, got $code)" && FAIL=$((FAIL + 1))
    fi
}

echo " TinyGate Smoke Test"
echo "══════════════════════════════════════"
echo ""

# --- Health ---
echo "── Health ──"
code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/health")
check "health check" "$code" 200

# --- Auth ---
echo "── Auth ──"
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $KEY" "$BASE/opencode/zen/go/v1/models")
check "valid key" "$code" 200

code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer sk-wrong" "$BASE/opencode/zen/go/v1/models")
check "invalid key" "$code" 401

code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/opencode/zen/go/v1/models")
check "no key" "$code" 401

# --- Routing ---
echo "── Routing ──"
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $KEY" "$BASE/opencode/zen/go/v1/models")
check "opencode route" "$code" 200

code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $KEY" "$BASE/zhipu/v4/models")
check "zhipu route" "$code" 200

code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $KEY" "$BASE/mimo/v1/models")
check "mimo route" "$code" 200

code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $KEY" "$BASE/nowhere")
check "404 unknown" "$code" 404

echo ""
echo "══════════════════════════════════════"
echo " $PASS passed, $FAIL failed"
echo "══════════════════════════════════════"

[ "$FAIL" -eq 0 ] && green "All gateway tests passed!" && exit 0
red "Some tests failed." && exit 1

#!/bin/bash
# TinyGate Demo — show request/response for each route
BASE="http://localhost:39901"
KEY="sk-gateway-key-1"

demo() {
    local name="$1" url="$2" body="$3"
    echo ""
    echo "══════════════════════════════════════════"
    echo "  $name"
    echo "══════════════════════════════════════════"
    echo ""
    echo "── Request ──"
    echo "POST $url"
    echo "$body" | python3 -m json.tool 2>/dev/null || echo "$body"
    echo ""
    echo "── Response ──"
    curl -s -X POST "$url" \
        -H "Authorization: Bearer $KEY" \
        -H "Content-Type: application/json" \
        -d "$body" | python3 -m json.tool 2>/dev/null || \
    curl -s -X POST "$url" \
        -H "Authorization: Bearer $KEY" \
        -H "Content-Type: application/json" \
        -d "$body"
    echo ""
}

# Models list (always works)
echo "══════════════════════════════════════════"
echo "  Models List"
echo "══════════════════════════════════════════"
echo ""
echo "── opencode ──"
curl -s -H "Authorization: Bearer $KEY" "$BASE/opencode/zen/go/v1/models" | python3 -c "import sys,json;d=json.load(sys.stdin);[print(f'  {m[\"id\"]}') for m in d['data'][:5]];print(f'  ... +{len(d[\"data\"])-5} more')"

# MiMo chat
demo "mimo-v2.5 chat" "$BASE/mimo/v1/chat/completions" \
    '{"model":"mimo-v2.5","messages":[{"role":"user","content":"say hello in one word"}],"max_tokens":20}'


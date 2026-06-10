#!/bin/bash
BASE="http://localhost:39901"

echo "=== Models ==="
curl -s $BASE/opencode/v1/models -H "Authorization: Bearer sk-gateway-key-1" | python3 -m json.tool 2>/dev/null | head -20

echo ""
echo "=== Chat ==="
curl -s $BASE/opencode/v1/chat/completions \
  -H "Authorization: Bearer sk-gateway-key-1" \
  -H "Content-Type: application/json" \
  -d '{"model":"deepseek-v4-flash","messages":[{"role":"user","content":"用一句话介绍中国"}],"max_tokens":100}' \
  | python3 -c "import sys,json;d=json.load(sys.stdin);print(d['choices'][0]['message']['content'])"

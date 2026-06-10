#!/bin/bash
# TinyGate 模型速查 — 列出所有可用模型及调用方式
BASE="http://localhost:39901"
KEY="sk-gateway-key-1"

cat << 'REF'
══════════════════════════════════════════════════════════════
  TinyGate 模型速查
══════════════════════════════════════════════════════════════

调用格式（所有路由统一）

  curl $BASE/{prefix}/{api_path} \
    -H "Authorization: Bearer $KEY" \
    -H "Content-Type: application/json" \
    -d '{模型相关的json数据}'

──────────────────────────────────────────────────────────────
  OpenCode Go      prefix: /opencode       18 models
──────────────────────────────────────────────────────────────
REF
curl -s $BASE/opencode/v1/models -H "Authorization: Bearer $KEY" \
  | python3 -c "
import sys,json
d=json.load(sys.stdin)
for m in sorted(d['data'],key=lambda x:x['id']):
    print(f'  {m[\"id\"]:<28s}')
"

cat << 'REF'

──────────────────────────────────────────────────────────────
  MiMo (小米)       prefix: /mimo          10 models
──────────────────────────────────────────────────────────────
REF
curl -s $BASE/mimo/v1/models -H "Authorization: Bearer $KEY" \
  | python3 -c "
import sys,json
d=json.load(sys.stdin)
for m in d['data']:
    print(f'  {m[\"id\"]:<28s}')
"

cat << 'REF'

──────────────────────────────────────────────────────────────
  智谱 (GLM)        prefix: /zhipu          7 models
──────────────────────────────────────────────────────────────
REF
curl -s $BASE/zhipu/v4/models -H "Authorization: Bearer $KEY" \
  | python3 -c "
import sys,json
d=json.load(sys.stdin)
for m in d['data']:
    print(f'  {m[\"id\"]:<28s}')
"

cat << 'REF'

══════════════════════════════════════════════════════════════
  curl 示例
══════════════════════════════════════════════════════════════

# OpenCode Go — DeepSeek
curl -s $BASE/opencode/v1/chat/completions \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"deepseek-v4-flash","messages":[{"role":"user","content":"你好"}],"max_tokens":100}'

# MiMo — 小米自研
curl -s $BASE/mimo/v1/chat/completions \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"mimo-v2.5","messages":[{"role":"user","content":"你好"}],"max_tokens":100}'

# 智谱 — GLM
curl -s $BASE/zhipu/v4/chat/completions \
  -H "Authorization: Bearer $KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"glm-4-flash","messages":[{"role":"user","content":"你好"}],"max_tokens":100}'
REF

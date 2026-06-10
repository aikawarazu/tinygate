# just-llm-gateway 设计文档

## 概述

一个纯轻量的 LLM 模型网关，用于屏蔽下游模型 API Key 的变动。客户端使用统一的 API Key 调用网关，网关将请求透明转发到下游模型提供商，并自动替换 Auth 信息。

**核心目标**：
- 屏蔽模型 API Key 变动：换 Key 时只需改网关配置，客户端代码无需改动
- 轻量：单二进制，零外部依赖，配置文件驱动
- 透明透传：不关心下游 API 结构，完全透传请求和响应

## 技术栈

- **语言**：Go
- **配置格式**：YAML
- **HTTP 框架**：标准库 `net/http`
- **反向代理**：标准库 `net/http/httputil.ReverseProxy`
- **部署方式**：单二进制 + Docker

## 架构设计

```
                     ┌─────────────────────────────────────┐
                     │         just-llm-gateway             │
                     │                                      │
   Client ──────────►│ 1. Auth Middleware                    │
   Bearer sk-xxx     │    └─ 匹配 api_keys 列表，任一通过    │
                     │                                      │
                     │ 2. Route Matcher                     │
                     │    └─ 按请求路径前缀匹配 routes 配置    │
                     │                                      │
                     │ 3. Request Rewrite (Director)         │
                     │    ├─ 剥离路径前缀                    │
                     │    ├─ 替换 Authorization 为下游 key   │
                     │    └─ 透传其余 Header + Body          │
                     │                                      │
                     │ 4. httputil.ReverseProxy ────────────►│  下游模型
                     │    ├─ 流式 SSE 透传                  │   deepseek
                     │    └─ 非流式 JSON 透传                │   anthropic
                     │                                      │   ...
                     │ 5. Response 透传回客户端              │
                     └─────────────────────────────────────┘
```

## 路由模型

**规则**：`downstream_url` 包含客户端路径之外的完整前缀。客户端用简单的标准化路径，网关自动补全下游完整路径。

**剥离逻辑**：
1. 匹配请求路径前缀（如 `/zhipu`）
2. 剥离前缀，得到剩余路径（如 `/v4/chat/completions`）
3. 拼接：`downstream_url` + 剩余路径 = 目标 URL

**示例**：
```
POST /zhipu/v4/chat/completions     → https://open.bigmodel.cn/api/paas/v4/chat/completions
POST /mimo/v1/chat/completions      → https://api.xiaomimimo.com/v1/chat/completions
POST /opencode/v1/chat/completions  → https://opencode.ai/zen/go/v1/chat/completions
```

## 配置文件设计

```yaml
# config.yaml
server:
  port: 39901
  timeout: 1200s          # 默认 1200s (20min)，支持 "1200s" / "20m" 格式
  health: true            # 默认开启 /health

gateway:
  api_keys:
    - "sk-my-key-1"
    - "sk-my-key-2"

routes:
  - prefix: "/zhipu"
    downstream_url: "https://open.bigmodel.cn/api/paas"
    api_key: "${ZHIPU_API_KEY}"
    # auth_header: "Authorization"         # 可选，默认值
    # auth_format: "Bearer ${api_key}"     # 可选，默认值

  - prefix: "/mimo"
    downstream_url: "https://api.xiaomimimo.com"
    api_key: "${MIMO_API_KEY}"

  - prefix: "/opencode"
    downstream_url: "https://opencode.ai/zen/go"
    api_key: "${OPENCODE_GO_API_KEY}"
```

### 环境变量注入

- `${VAR_NAME}`：加载时替换为环境变量值
- `$${NOT_VAR}`：双写 `$` 转义，不替换

### 默认路由

提供三个默认路由，覆盖主流中国 LLM 提供商：

| 提供商 | prefix | downstream_url | auth 格式 |
|--------|--------|----------------|-----------|
| 智谱 (GLM) | `/zhipu` | `https://open.bigmodel.cn/api/paas` | `Authorization: Bearer` |
| 小米 (MiMo) | `/mimo` | `https://api.xiaomimimo.com` | `Authorization: Bearer` |
| OpenCode Go | `/opencode` | `https://opencode.ai/zen/go` | `Authorization: Bearer` |

## 认证机制

### 上游认证（客户端 → 网关）

- 支持多个 API Key，任意一个均可通过
- 验证 `Authorization: Bearer sk-xxx` 中的 token 是否在 `api_keys` 列表中
- 401 Unauthorized 如果 token 无效

### 下游认证（网关 → 模型提供商）

- 默认：`Authorization: Bearer ${api_key}`
- 可选覆盖：`auth_header` 和 `auth_format` 字段
- 90% 的厂商用 Bearer Auth，少数例外（如 Anthropic 用 `x-api-key`）可通过可选字段覆盖

## Header 处理策略

- **默认全透传**：除 `Authorization` 和 `Host` 外，所有 Header 原样转发
- **Authorization**：替换为下游 API Key（按 `auth_format` 模板）
- **Host**：自动剥离（避免下游收到错误的 Host）

## 流式响应支持

- 自动检测请求中的 `stream` 字段
- 流式响应（SSE）：原样透传 `Transfer-Encoding: chunked`
- 非流式响应：完整 JSON 透传
- 使用 `httputil.ReverseProxy` 原生支持，无需额外处理

## 超时配置

- 默认超时：1200 秒（20 分钟）
- 全局配置，所有路由共享
- 超时返回 504 Gateway Timeout
- 不支持重试（避免重复费用）

## 日志记录

- 记录请求摘要：方法、路径、状态码、耗时
- 不记录请求/响应 Body（避免泄漏敏感数据）
- 输出到 stdout

## 健康检查

- `GET /health` 返回 200 OK
- 不走认证和代理逻辑

## 代码结构

```
just-llm-gateway/
├── main.go              # 入口，启动 server
├── config/
│   └── config.go        # YAML 配置解析 + 环境变量注入
├── gateway/
│   ├── auth.go          # Auth 中间件，验证 api_keys
│   ├── router.go        # 路由匹配（前缀匹配 + 剥离）
│   └── proxy.go         # ReverseProxy 封装，Director 改写
├── config.yaml          # 默认配置文件
├── Dockerfile           # Docker 构建
├── go.mod
└── go.sum
```

### 职责划分

- `config/config.go`：解析 YAML，`${VAR}` 替换，默认值
- `gateway/auth.go`：中间件：验证 `Authorization: Bearer` 是否在 `api_keys` 列表中
- `gateway/router.go`：按前缀最长匹配选路由，剥离前缀，构造目标 URL
- `gateway/proxy.go`：创建 `httputil.ReverseProxy`：Director 改请求、Transport 设超时、ErrorHandler 记录错误
- `main.go`：组装、启动 `http.Server`

## 错误处理

- 401 Unauthorized：上游认证失败
- 404 Not Found：无匹配路由
- 502 Bad Gateway：下游连接失败
- 504 Gateway Timeout：下游超时

## 部署

### 二进制

```bash
go build -o just-llm-gateway .
./just-llm-gateway -config config.yaml
```

### Docker

```bash
docker build -t just-llm-gateway .
docker run -p 39901:39901 -v ./config.yaml:/app/config.yaml just-llm-gateway
```

## 设计决策

### ADR-001：选择标准库而非框架

**决策**：使用 Go 标准库 `net/http` 和 `net/http/httputil.ReverseProxy`，不引入 gin/fiber 等框架。

**理由**：
- 轻量：零第三方依赖，二进制约 8MB
- `ReverseProxy` 原生支持 SSE 流式透传
- Director 函数灵活修改请求
- 标准库足够满足需求

### ADR-002：路由模型选择

**决策**：`downstream_url` 包含客户端路径之外的完整前缀。客户端用简单的标准化路径，网关自动补全下游完整路径。

**理由**：
- 客户端路径简单：统一用 `/v1/chat/completions`，不用记每个厂商的 API 路径结构
- 配置灵活：`downstream_url` 可以包含任意长度的路径前缀
- 逻辑简单：剥离前缀，剩余路径直接拼到 downstream_url
- 真正的透明透传：网关不关心下游 API 结构

### ADR-003：不支持重试

**决策**：网关不支持自动重试。

**理由**：
- LLM 调用可能产生费用，重试可能导致重复计费
- 简化网关逻辑
- 让客户端自己决定是否重试

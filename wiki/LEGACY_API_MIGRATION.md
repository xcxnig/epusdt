# Legacy API 兼容性变更文档

## 概述

为了适配原 dujiaoka 插件，本次更新将所有传统 API 路由迁移到 `/legacy` 前缀下，并添加了向后兼容的默认值处理。

## 后台配置说明（重要）

### 对于 dujiaoka 用户

**只需要修改一个地方**：在 dujiaoka 后台支付插件配置中，将 API 地址前缀从 `/api` 改为 `/legacy/api` 即可。

**示例**：

```
旧配置：https://your-domain.com/api/v1/order/create-transaction
新配置：https://your-domain.com/legacy/api/v1/order/create-transaction
```

**就这么简单！** 其他配置（密钥、回调地址等）完全不需要修改。

---

## 路由变更对照表

### 支付相关路由

| 原路由 | 新路由 | 说明 |
|--------|--------|------|
| `GET /pay/checkout-counter/:trade_id` | `GET /legacy/pay/checkout-counter/:trade_id` | 收银台页面 |
| `GET /pay/check-status/:trade_id` | `GET /legacy/pay/check-status/:trade_id` | 支付状态检测 |

### API 路由

| 原路由 | 新路由 | 说明 |
|--------|--------|------|
| `POST /api/v1/order/create-transaction` | `POST /legacy/api/v1/order/create-transaction` | 创建交易订单 |

## 代码变更详情

### 1. 路由前缀变更

**文件**: `src/route/router.go`

```go
// 旧版本
payRoute := e.Group("/pay")
apiV1Route := e.Group("/api/v1")

// 新版本
payRoute := e.Group("/legacy/pay")
apiV1Route := e.Group("/legacy/api/v1")
```

### 2. 控制器方法重命名

所有 legacy API 相关的控制器方法都添加了 `Legecy` 前缀：

| 原方法名 | 新方法名 |
|---------|---------|
| `CreateTransaction` | `LegecyCreateTransaction` |
| `CheckoutCounter` | `LegecyCheckoutCounter` |
| `CheckStatus` | `LegecyCheckStatus` |

### 3. 中间件重命名

**文件**: `src/middleware/check_sign.go`

```go
// 旧版本
func CheckApiSign() echo.MiddlewareFunc

// 新版本
func LegecyCheckApiSign() echo.MiddlewareFunc
```

### 4. 创建订单接口兼容性增强

**文件**: `src/controller/comm/order_controller.go`

为了兼容旧版本 dujiaoka 插件，在 `LegecyCreateTransaction` 方法中添加了默认值处理：

```go
// 兼容旧版本插件：如果未传递 token 和 currency，使用默认值
if req.Token == "" {
    req.Token = "usdt"
}
if req.Currency == "" {
    req.Currency = "cny"
}
```

**影响**：
- 如果请求中未提供 `token` 参数，默认使用 `usdt`
- 如果请求中未提供 `currency` 参数，默认使用 `cny`

### 5. 收银台页面数据适配

**文件**: `src/controller/comm/pay_controller.go`

```go
resp.Token = resp.ReceiveAddress // only for legacy checkout counter, token is the receive address
```

**说明**：为了兼容旧版收银台模板，将 `ReceiveAddress` 字段映射到 `Token` 字段。

### 6. 前端页面更新

**文件**: `src/static/index.html`

```javascript
// 旧版本
url: "/pay/check-status/{{.TradeId}}"

// 新版本
url: "/legacy/pay/check-status/{{.TradeId}}"
```

## 迁移注意事项

### 对于 dujiaoka 插件开发者

1. **更新所有 API 端点**：将所有请求 URL 从 `/api/v1/*` 更新为 `/legacy/api/v1/*`
2. **更新支付页面 URL**：将 `/pay/*` 更新为 `/legacy/pay/*`
3. **可选参数**：`token` 和 `currency` 参数现在是可选的，系统会自动填充默认值

### 对于新版本 API

- 新版本 API（非 legacy）应使用不带 `/legacy` 前缀的路由
- 新版本 API 要求严格传递所有必需参数，不提供默认值

## 兼容性保证

- ✅ 旧版 dujiaoka 插件无需修改即可使用（通过默认值兼容）
- ✅ 所有 legacy 路由保持功能完整性
- ✅ 签名验证机制保持不变
- ✅ 收银台页面模板兼容

---

## 新增配置项说明

本次更新新增了以下配置项（位于 `.env` 文件）：

### 1. `api_rate_url` - 汇率接口 URL

用于获取实时汇率的 API 地址。系统会根据此接口动态获取不同币种的汇率。

```bash
# 汇率接口url
api_rate_url=https://your-rate-api.com/
```

**API 格式要求**：

系统会请求 `{api_rate_url}/{currency}.json`，例如：
- `https://your-rate-api.com/cny.json`
- `https://your-rate-api.com/usd.json`

**返回格式示例**：

```json
{
  "cny": {
    "usdt": 0.1389,
    "trx": 0.0123
  }
}
```

其中 `0.1389` 表示 1 CNY = 0.1389 USDT（即 1 USDT ≈ 7.2 CNY）

**说明**：
- 支持自建汇率 API，只需按照上述格式返回数据即可

### 2. `tron_grid_api_key` - TRON Grid API Key

TRON Grid API 密钥，用于提高 API 请求限制和稳定性。

```bash
tron_grid_api_key=
```

**如何获取 API Key**：

1. 访问 [https://www.trongrid.io/](https://www.trongrid.io/)
2. 注册账号并登录
3. 在控制台创建 API Key
4. 将 API Key 填入配置文件

**为什么需要 API Key**：

- ✅ **提高请求限制**：免费账号有更高的 API 调用配额
- ✅ **更好的稳定性**：避免公共 API 的限流问题
- ✅ **支持更多功能**：为后续支持 TRX 等其他代币做准备

### 配置示例

完整的 `.env` 配置示例：

```bash
# 订单过期时间（分钟）
order_expiration_time=10

# 订单回调失败最大重试次数
order_notice_max_retry=0

# 汇率接口url（动态获取汇率）
api_rate_url=

# TRON Grid API Key（推荐配置）
tron_grid_api_key=your-api-key-here
```

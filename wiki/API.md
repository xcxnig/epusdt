# Epusdt API 文档

开发者可通过 Epusdt 提供的 HTTP API 将收款能力集成到业务系统。本文档以当前代码路由为准。

> 旧版 `POST /api/v1/order/create-transaction` 已不再注册；创建订单请使用 `POST /payments/gmpay/v1/order/create-transaction`。

## 接口总览

| 场景 | 方法 | 路径 | 是否需要签名 |
| --- | --- | --- | --- |
| 创建 GMPay 交易 | POST | `/payments/gmpay/v1/order/create-transaction` | 是 |
| 获取公开支付配置 | GET | `/payments/gmpay/v1/config` | 否 |
| 收银台页面 | GET | `/pay/checkout-counter/{trade_id}` | 否 |
| 收银台初始化数据 | GET | `/pay/checkout-counter-resp/{trade_id}` | 否 |
| 查询支付状态 | GET | `/pay/check-status/{trade_id}` | 否 |
| 切换支付网络/通道 | POST | `/pay/switch-network` | 否 |
| EPay 兼容创建交易 | GET/POST | `/payments/epay/v1/order/create-transaction/submit.php` | 是 |
| OkPay 平台回调 | POST | `/payments/okpay/v1/notify` | OkPay 签名 |

## 统一响应格式

除重定向和纯文本回调接口外，接口返回 JSON：

```json
{
  "status_code": 200,
  "message": "success",
  "data": {},
  "request_id": "b1344d70-ff19-4543-b601-37abfb3b3686"
}
```

说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `status_code` | integer | 业务状态码。成功为 `200`，错误码见文末。 |
| `message` | string | 返回消息。 |
| `data` | object/null | 接口数据。 |
| `request_id` | string | 请求 ID，服务端自动生成。 |

签名错误会返回 HTTP 401；业务错误通常返回 HTTP 400，并在 `status_code` 中给出具体业务码。

## 签名规则

当前版本使用统一商户凭证。请求必须携带 `pid`，服务端用 `pid` 查询对应的 `secret_key` 作为签名密钥。默认安装会创建一个 PID 为 `1000` 的默认密钥。

### GMPay 签名

1. 将所有非空参数按参数名 ASCII 字典序升序排序。
2. 使用 `key=value` 形式以 `&` 拼接。
3. 不参与签名的字段：`signature`。
4. 在拼接字符串末尾直接追加 `secret_key`。
5. 对最终字符串做 MD5，结果转小写，作为 `signature`。

注意：

- `pid` 必须参与签名。
- 空字符串和 `null` 不参与签名。
- 参数名区分大小写。
- JSON 数字会按服务端数字格式参与签名，例如 `100.00` 会被解析为 `100`；如果需要保留字符串格式，可使用 `application/x-www-form-urlencoded`。

示例参数：

```text
pid=1000
order_id=ORD202605230001
currency=cny
token=usdt
network=tron
amount=100
notify_url=https://merchant.example/notify
redirect_url=https://merchant.example/return
name=VIP
```

以下示例假设 `secret_key` 为 `epusdt_secret_key`，仅用于演示签名计算。

待签名字符串：

```text
amount=100&currency=cny&name=VIP&network=tron&notify_url=https://merchant.example/notify&order_id=ORD202605230001&pid=1000&redirect_url=https://merchant.example/return&token=usdtepusdt_secret_key
```

得到：

```text
signature=476412c422f4dd75c3d533f5c47a9cac
```

### PHP 签名示例

GMPay 使用 `signature` 字段，签名时只排除 `signature`：

```php
function gmpaySign(array $params, string $secretKey): string
{
    unset($params['signature']);
    ksort($params, SORT_STRING);

    $pairs = [];
    foreach ($params as $key => $value) {
        if ($value === '' || $value === null) {
            continue;
        }
        $pairs[] = $key . '=' . $value;
    }

    return strtolower(md5(implode('&', $pairs) . $secretKey));
}
```

EPay 兼容接口使用 `sign` 字段，签名时排除 `sign` 和 `sign_type`：

```php
function epaySign(array $params, string $secretKey): string
{
    unset($params['sign'], $params['sign_type']);
    ksort($params, SORT_STRING);

    $pairs = [];
    foreach ($params as $key => $value) {
        if ($value === '' || $value === null) {
            continue;
        }
        $pairs[] = $key . '=' . $value;
    }

    return strtolower(md5(implode('&', $pairs) . $secretKey));
}
```

## 创建 GMPay 交易

`POST /payments/gmpay/v1/order/create-transaction`

支持：

- `Content-Type: application/json`
- `Content-Type: application/x-www-form-urlencoded`

### 请求示例

```json
{
  "pid": "1000",
  "order_id": "ORD202605230001",
  "currency": "cny",
  "token": "usdt",
  "network": "tron",
  "amount": 100,
  "notify_url": "https://merchant.example/notify",
  "redirect_url": "https://merchant.example/return",
  "name": "VIP",
  "signature": "476412c422f4dd75c3d533f5c47a9cac"
}
```

### 请求参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `pid` | string | 是 | 商户 PID，用于查找 API Key，并参与签名。 |
| `order_id` | string | 是 | 商户订单号，最长 32 字符，不能重复。 |
| `currency` | string | 是 | 法币币种，如 `cny`、`usd`。 |
| `token` | string | 是 | 收款币种，如 `usdt`、`trx`、`usdc`、`sol`。 |
| `network` | string | 是 | 收款网络，如 `tron`、`solana`、`ethereum`、`bsc`、`polygon`、`plasma`。 |
| `amount` | number | 是 | 法币金额，必须大于 `0.01`。 |
| `notify_url` | string | 是 | 支付成功异步回调地址。 |
| `redirect_url` | string | 否 | 支付完成后的同步跳转地址。 |
| `name` | string | 否 | 商品/订单名称。 |
| `payment_type` | string | 否 | 兼容字段。普通 GMPay 不需要传；传 `Epay` 会使用 EPay 回调格式，且 PID 必须是数字。 |
| `signature` | string | 是 | GMPay 签名。 |

建议先调用 `/payments/gmpay/v1/config` 获取可用的 `network` 和 `token` 组合。

### 成功响应

```json
{
  "status_code": 200,
  "message": "success",
  "data": {
    "trade_id": "20260523171652123456001",
    "order_id": "ORD202605230001",
    "amount": 100,
    "currency": "CNY",
    "actual_amount": 14.29,
    "receive_address": "TTestTronAddress001",
    "token": "USDT",
    "expiration_time": 1779530812,
    "payment_url": "https://pay.example.com/pay/checkout-counter/20260523171652123456001"
  },
  "request_id": "b1344d70-ff19-4543-b601-37abfb3b3686"
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `trade_id` | string | Epusdt 交易号。 |
| `order_id` | string | 商户订单号。 |
| `amount` | number | 商户提交的法币金额。 |
| `currency` | string | 法币币种。 |
| `actual_amount` | number | 实际需支付的加密货币数量。 |
| `receive_address` | string | 收款地址。 |
| `token` | string | 收款币种。 |
| `expiration_time` | integer | 订单过期时间，秒级时间戳。 |
| `payment_url` | string | 收银台地址。该地址会跳转到前端收银台。 |

## 获取公开支付配置

`GET /payments/gmpay/v1/config`

返回收银台展示配置、可用链/币种、EPay 默认配置和 OkPay 公共配置。

### 成功响应示例

```json
{
  "status_code": 200,
  "message": "success",
  "data": {
    "supported_assets": [
      {
        "network": "tron",
        "display_name": "TRON",
        "tokens": ["TRX", "USDT"]
      },
      {
        "network": "solana",
        "display_name": "Solana",
        "tokens": ["SOL", "USDC", "USDT"]
      }
    ],
    "site": {
      "cashier_name": "Acme Cashier",
      "logo_url": "https://cdn.example.com/logo.png",
      "website_title": "Acme Payments",
      "support_link": "https://example.com/support",
      "background_color": "#0f172a",
      "background_image_url": "https://cdn.example.com/background.png"
    },
    "epay": {
      "default_token": "usdt",
      "default_currency": "cny",
      "default_network": "tron"
    },
    "okpay": {
      "enabled": false,
      "allow_tokens": ["USDT", "TRX"]
    },
    "version": "v1.0.1"
  },
  "request_id": "b1344d70-ff19-4543-b601-37abfb3b3686"
}
```

`supported_assets` 只包含同时满足以下条件的组合：

- 链已启用。
- 该链有可用钱包地址。
- 该链至少有一个启用中的 token。

## 收银台页面

`GET /pay/checkout-counter/{trade_id}`

用于浏览器打开收银台。当前实现会返回 301，并跳转到：

```text
/cashier/{trade_id}
```

创建交易接口返回的 `payment_url` 即为该地址。

## 收银台初始化数据

`GET /pay/checkout-counter-resp/{trade_id}`

用于前端收银台读取订单展示数据。该接口只确认订单存在并返回基础数据；当前支付状态请调用 `/pay/check-status/{trade_id}`。

### 成功响应示例

```json
{
  "status_code": 200,
  "message": "success",
  "data": {
    "trade_id": "20260523171652123456001",
    "amount": 100,
    "actual_amount": 14.29,
    "token": "USDT",
    "currency": "CNY",
    "receive_address": "TTestTronAddress001",
    "network": "tron",
    "expiration_time": 1779530812000,
    "redirect_url": "https://merchant.example/return",
    "payment_url": "",
    "created_at": 1779530212000,
    "is_selected": false
  },
  "request_id": "b1344d70-ff19-4543-b601-37abfb3b3686"
}
```

注意：该接口的 `expiration_time` 和 `created_at` 是毫秒级时间戳。

## 查询支付状态

`GET /pay/check-status/{trade_id}`

### 成功响应示例

```json
{
  "status_code": 200,
  "message": "success",
  "data": {
    "trade_id": "20260523171652123456001",
    "status": 1
  },
  "request_id": "b1344d70-ff19-4543-b601-37abfb3b3686"
}
```

订单状态：

| 值 | 说明 |
| --- | --- |
| `1` | 等待支付 |
| `2` | 支付成功 |
| `3` | 已过期 |

## 切换支付网络/通道

`POST /pay/switch-network`

该接口通常由收银台前端调用，用于切换到另一个链上收款地址，或切换到 OkPay 托管收银台。

### 请求示例

```json
{
  "trade_id": "20260523171652123456001",
  "token": "USDT",
  "network": "solana"
}
```

切换到 OkPay：

```json
{
  "trade_id": "20260523171652123456001",
  "token": "USDT",
  "network": "okpay"
}
```

### 请求参数

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `trade_id` | string | 是 | 父订单交易号。 |
| `token` | string | 是 | 目标币种。 |
| `network` | string | 是 | 目标网络，或特殊值 `okpay`。 |

### 成功响应

返回结构与收银台初始化数据一致。链上子订单的 `payment_url` 通常是本地收银台地址；OkPay 子订单的 `payment_url` 是 OkPay 返回的托管支付链接。

说明：

- 只能对父订单切换网络，不能对子订单继续切换。
- 父订单必须仍处于等待支付状态。
- 每个父订单最多创建 2 个等待支付中的子订单。
- 如果切换到同一组 `token + network`，会返回已有订单。

## EPay 兼容创建交易

`GET /payments/epay/v1/order/create-transaction/submit.php`

`POST /payments/epay/v1/order/create-transaction/submit.php`

该接口兼容传统 EPay/易支付接入方式。成功后不会返回 JSON，而是 HTTP 302 跳转到：

```text
/pay/checkout-counter/{trade_id}
```

### 请求参数

| 字段 | 位置 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- | --- |
| `pid` | query/form | string | 是 | 商户 PID。建议使用数字 PID；EPay 回调会按数字 PID 输出。 |
| `money` | query/form | number | 是 | 法币金额。 |
| `out_trade_no` | query/form | string | 是 | 商户订单号。 |
| `notify_url` | query/form | string | 是 | 异步回调地址。 |
| `return_url` | query/form | string | 否 | 支付完成后的同步跳转地址。 |
| `name` | query/form | string | 否 | 商品/订单名称。 |
| `type` | query/form | string | 否 | 兼容字段，如 `alipay`。创建订单时不决定实际链上币种。 |
| `sign` | query/form | string | 是 | EPay 签名。 |
| `sign_type` | query/form | string | 否 | 通常为 `MD5`。 |

签名规则：

- 使用 `pid` 对应的 `secret_key`。
- 排除 `sign` 和 `sign_type`。
- 其他非空参数按 ASCII 字典序拼接后追加 `secret_key` 并 MD5；如果接入插件额外传了 `sitename` 等字段，也要一起参与签名。

示例待签名字符串：

```text
money=100&name=VIP&notify_url=https://merchant.example/notify&out_trade_no=ORD202605230001&pid=1000&return_url=https://merchant.example/return&type=alipayepusdt_secret_key
```

得到：

```text
sign=b865b0acbb2b01554c35a1bd33351452
```

EPay 接口会使用后台配置的默认 `token`、`currency`、`network` 创建实际订单，默认配置可通过 `/payments/gmpay/v1/config` 的 `epay` 字段查看。

## 商户异步回调

订单支付成功后，Epusdt 会向订单的 `notify_url` 发送异步通知。目标服务器处理完成后需返回 HTTP 200，响应体为 `ok` 或 `success`（大小写不敏感）。否则会按队列配置重试：首次失败后最多重试 `order_notice_max_retry` 次，重试间隔按 `callback_retry_base_seconds` 指数退避，最大 5 分钟。

### GMPay 回调

普通 GMPay 订单使用 POST JSON 回调。

```json
{
  "pid": "1000",
  "trade_id": "20260523171652123456001",
  "order_id": "ORD202605230001",
  "amount": 100,
  "actual_amount": 14.29,
  "receive_address": "TTestTronAddress001",
  "token": "USDT",
  "block_transaction_id": "0xabc123...",
  "signature": "a1b2c3d4e5f6...",
  "status": 2
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `pid` | string | 订单所属 API Key 的 PID。商户应使用该 PID 查本地密钥验签。 |
| `trade_id` | string | Epusdt 交易号。 |
| `order_id` | string | 商户订单号。 |
| `amount` | number | 商户提交的法币金额。 |
| `actual_amount` | number | 实际到账的加密货币数量。 |
| `receive_address` | string | 收款地址。 |
| `token` | string | 收款币种。 |
| `block_transaction_id` | string | 链上交易哈希或第三方支付订单号。 |
| `signature` | string | 回调签名。 |
| `status` | integer | 当前仅支付成功时回调，值为 `2`。 |

GMPay 回调验签方式与创建订单一致，但排除 `signature` 字段。

### EPay 兼容回调

通过 EPay 兼容接口创建的订单，会使用 GET 请求回调 `notify_url`，参数如下：

> EPay 回调会把 `pid` 输出为数字；使用 EPay 兼容接口或 `payment_type=Epay` 时，请确保 API Key 的 PID 是数字。

```text
pid=1000
trade_no=20260523171652123456001
out_trade_no=ORD202605230001
type=alipay
name=VIP
money=100.0000
trade_status=TRADE_SUCCESS
sign=a1b2c3d4...
sign_type=MD5
```

验签时排除 `sign` 和 `sign_type`，其余非空参数按 ASCII 字典序拼接后追加 `secret_key` 并 MD5。

## OkPay 平台回调

`POST /payments/okpay/v1/notify`

这是 OkPay/OkayPay 平台通知 Epusdt 的接口，不是商户系统主动调用的接口。配置 OkPay 时，回调地址应填写该路径。

支持 JSON、`application/x-www-form-urlencoded`、multipart form 和原始 query-string 风格 body。成功返回纯文本：

```text
success
```

失败返回 HTTP 400：

```text
fail
```

Epusdt 会按配置的 OkPay shop token 验证 OkPay 签名，成功后将对应 OkPay 子订单标记为已支付，并触发父订单商户回调。

## status_code 返回状态码及含义

| 状态码 | HTTP 状态 | 说明 |
| --- | --- | --- |
| `200` | 200 | 成功 |
| `400` | 400 | 系统错误，或普通参数/验证错误 |
| `401` | 401 | 签名认证错误 |
| `10001` | 400 | 钱包地址已存在 |
| `10002` | 400 | 支付交易已存在，请勿重复创建 |
| `10003` | 400 | 无可用钱包地址，无法发起支付 |
| `10004` | 400 | 支付金额有误，无法满足最小支付单位 |
| `10005` | 400 | 无可用金额通道 |
| `10006` | 400 | 汇率计算错误 |
| `10007` | 400 | 订单区块已处理 |
| `10008` | 400 | 订单不存在 |
| `10009` | 400 | 无法解析参数 |
| `10010` | 400 | 订单状态已变化 |
| `10011` | 400 | 超过子订单数量上限 |
| `10012` | 400 | 不能对子订单切换网络 |
| `10013` | 400 | 订单不是等待支付状态 |
| `10014` | 400 | 链未启用 |
| `10016` | 400 | 支持的资产不存在 |
| `10017` | 400 | 支付服务商未启用 |
| `10018` | 400 | 支付服务商配置不完整 |
| `10019` | 400 | 支付服务商不支持该币种或网络 |

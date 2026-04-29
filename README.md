# Epusdt — Easy Payment USDT

<p align="center">
  <img src="wiki/img/usdtlogo.png" alt="Epusdt Logo - Multi-chain Crypto Payment Gateway" width="120">
</p>

<p align="center">
  <strong>开源多链多币种 Crypto 支付网关 · 实际采用率 Top 1</strong>
</p>

<p align="center">
  <a href="https://epusdt.com"><img src="https://img.shields.io/badge/官网文档-epusdt.com-blue?style=for-the-badge" alt="Official Docs"></a>
  <a href="https://t.me/epusdt"><img src="https://img.shields.io/badge/Telegram-频道-26A5E4?style=for-the-badge&logo=telegram&logoColor=white" alt="Telegram Channel"></a>
  <a href="https://t.me/epusdt_group"><img src="https://img.shields.io/badge/Telegram-交流群-26A5E4?style=for-the-badge&logo=telegram&logoColor=white" alt="Telegram Group"></a>
</p>

<p align="center">
  <a href="https://github.com/GMWalletApp/epusdt/stargazers"><img src="https://img.shields.io/github/stars/GMWalletApp/epusdt?style=flat-square&color=f5c542" alt="GitHub Stars 3000+"></a>
  <a href="https://www.gnu.org/licenses/gpl-3.0.html"><img src="https://img.shields.io/badge/License-GPLv3-blue?style=flat-square" alt="GPLv3 License"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.16+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.16+"></a>
  <a href="https://github.com/GMWalletApp/epusdt/releases"><img src="https://img.shields.io/github/v/release/GMWalletApp/epusdt?style=flat-square&color=green" alt="Latest Release"></a>
</p>

---

## 🌍 What is Epusdt?

**Epusdt** (Easy Payment USDT) is a self-hosted **multi-chain, multi-token crypto payment gateway** built with Go. It has evolved from a TRC20-only solution into a comprehensive **multi-chain receiving platform**, enabling any website or application to accept crypto payments across multiple blockchain networks and token types. No third-party fees, no custodial risk — funds go directly into your wallet.

> **⭐ GitHub Star 3000+** · **🔌 已支持站点解决方案 10+** · **🏆 Crypto 支付工具实际采用率 Top 1**

Deploy it privately, integrate via HTTP API, and start receiving **crypto payments** in minutes. That's it. 🎉

### 🔗 已支持网络与代币

| 网络 | 代币 |
|------|------|
| **TRC20** (Tron) | USDT、TRX|
| **ERC20** (Ethereum) | USDT、USDC、ETH |
| **Solana** | USDT、USDC |
| **BEP20** (BSC) | USDT、USDC、BNB |
| **Polygon** | USDT、USDC |
| **更多** | 持续扩展中… |

> 💡 具体支持的链与代币以 [最新版本](https://github.com/GMWalletApp/epusdt/releases) 及 [官方文档](https://epusdt.com) 为准。

---

## 🔌 广泛兼容，即插即用

无论你运营的是哪类系统，Epusdt 均可基于现有接口方案，**无需重构业务逻辑**，快速接入，立即获得 Crypto 收款能力，低成本扩展全球支付场景：

| 领域 | 已支持系统 |
|------|-----------|
| **AI 分发** | [OneAPI](https://github.com/songquanpeng/one-api)、[NewAPI](https://github.com/QuantumNous/new-api) |
| **发卡系统** | [独角数卡（Dujiaoka）](https://dujiao-next.com/)、[异次元发卡](https://github.com/lizhipay/acg-faka) |
| **代理面板** | [V2Board](https://github.com/v2board/v2board)、[XBoard](https://github.com/cedar2025/Xboard)、[xiaoV2board](https://github.com/wyx2685/v2board/)、[SSPanel](https://github.com/anankke/sspanel-uim) |
| **建站生态** | [WordPress](https://wordpress.com/)、[WHMCS](https://www.whmcs.com/) |
| **Epay兼容** | 兼容各类支持Epay易支付接口的平台 |
| **更多** | 简易HTTP API 10分钟内接入 |
 
👉 查看更多集成列表与插件：[plugins/](plugins/)

---

## ✨ 核心特性

- **多链多币种** — 支持 TRC20、ERC20、BEP20、Polygon 等主流网络，不再局限于单一链
- **私有化部署** — 无需担心钱包被篡改、吞单，资金完全自主掌控
- **零依赖运行** — 单个二进制文件即可启动，低并发场景无需安装 MySQL + Redis,一键部署零成本维护
- **跨平台** — 支持 x86 / ARM 架构的 Windows / Linux / Mac 设备
- **多钱包轮询** — 自动轮换收款地址，提高订单并发处理能力
- **异步队列** — 高性能消息回调，优雅处理高并发场景
- **HTTP API** — 标准化接口，任何语言 / 框架均可10min内集成
- **Telegram Bot** — 实时支付通知，快捷管理与监控

---

## 📖 文档与教程

完整文档请访问 👉 **[epusdt.com](https://epusdt.com)**

快速入门：

| 教程 | 说明 |
|------|------|
| [Docker 部署](https://epusdt.com/guide/installation/docker) | 推荐方式，一键启动 |
| [宝塔面板部署](https://epusdt.com/guide/installation/aapanel) | 适合宝塔用户 |
| [手动部署](https://epusdt.com/guide/installation/manual.html) | 完全手动控制 |
| [开发者 API 文档](https://epusdt.com/zh/guide/integration/gmpay.html) | 接口集成指南 |

---

## 🏗️ 项目结构

```
Epusdt
├── plugins/    → 已集成的系统插件（独角数卡等）
├── src/        → 项目核心代码
├── sdk/        → 接入 SDK
├── sql/        → 数据库安装 / 升级脚本
└── wiki/       → 文档与知识库
```

---

## 🔧 实现原理

Epusdt 通过监听多条区块链网络（TRC20、ERC20、BEP20、Polygon 等）的 API 或RPC节点，实时捕获钱包地址的代币入账事件，利用**金额差异**与**时效性**精确匹配交易归属：

```
工作流程：
1. 客户发起支付，需支付 20.05 USDT
2. 系统在哈希表中查找可用的 钱包地址 + 金额 组合
3. 若 address_1:20.05 未被占用 → 锁定该组合（有效期 10 分钟），返回给客户
4. 若已被占用 → 自动累加 0.0001 尝试下一个金额组合（最多 100 次）
5. 后台线程持续监听所有钱包的入账事件，金额匹配则确认支付成功
```

![Epusdt 支付流程图](wiki/img/implementation_principle.jpg)

---

## 💬 社区与支持

**遇到问题？** 请优先在 GitHub 提交 [Issue](https://github.com/GMWalletApp/epusdt/issues)，我们会**优先处理** Issue 中的反馈。

加入 Telegram 社区，获取最新开发动态，参与需求调研，与超过5000名活跃用户交流使用经验，对接商业资源：

| 渠道 | 链接 |
|------|------|
| 📢 **Epusdt 频道** | [https://t.me/epusdt](https://t.me/epusdt) |
| 💬 **Epusdt 交流群** | [https://t.me/epusdt_group](https://t.me/epusdt_group) |
| 📚 **官方文档站** | [https://epusdt.com](https://epusdt.com) |

---

## Star History

<a href="https://www.star-history.com/?type=date&repos=gmwalletapp%2Fepusdt">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/chart?repos=gmwalletapp/epusdt&type=date&theme=dark&legend=top-left" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/chart?repos=gmwalletapp/epusdt&type=date&legend=top-left" />
   <img alt="Star History Chart" src="https://api.star-history.com/chart?repos=gmwalletapp/epusdt&type=date&legend=top-left" />
 </picture>
</a>

---

## 📜 开源协议

Epusdt 遵守 [GPLv3](https://www.gnu.org/licenses/gpl-3.0.html) 开源协议。

---

## ⚠️ 免责声明

本项目仅供学习与技术交流使用，用户在使用过程中需自行遵守所在地法律法规。由于涉及加密资产及资金安全，用户应自行审查相关代码与风险。加密资产属于高风险新兴资产（包括稳定币），其价值可能波动甚至归零，GM Wallet 不对任何资产或使用结果作出保证。本内容不构成任何投资、税务、法律或金融建议，所有代码均按原样提供，仅供教育用途，相关决策请咨询专业人士。

---

<p align="center">
  <sub>
    <b>Keywords:</b> USDT Payment Gateway · Crypto Payment · Multi-chain Payment · TRC20 Payment · ERC20 Payment · BEP20 Payment · 
    Self-hosted Crypto Gateway · OneAPI Payment · NewAPI Payment · 独角数卡支付 · 异次元发卡支付方式 · 
    V2Board Payment · XBoard Payment · SSPanel 支付接口 · 
    WordPress Crypto Payment · WHMCS USDT Payment · Polygon USDT · 
    Epusdt · Easy Payment USDT · Open Source Payment Gateway · 多链收款
  </sub>
</p>

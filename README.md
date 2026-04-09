# evm-grid-bot

🤖 基于 Telegram 的 EVM 链网格交易机器人

<img width="538" height="676" alt="image" src="https://github.com/user-attachments/assets/ede5635a-16f5-4a78-940e-5ff01bbbc00c" />

## 📋 项目简介

EVMGridBot 是一个智能的网格交易机器人，通过 Telegram 接口为用户提供 EVM 链（目前支持 BSC 链，扩展到其他 EVM 会非常容易）上代币的自动化网格交易服务。用户只需设置价格区间，机器人将在该区间内自动执行低买高卖的网格交易策略，帮助用户在震荡行情中获利。

> 💡 提示：此项目专为 EVM 链设计。对于 Solana 链交易，推荐使用 [TP Bot](https://t.me/follow_step_bot?start=cwqTcEV3)，[使用教程](https://tpbot-2.gitbook.io/tpbot/ce-le-xin-shou-jiao-cheng/wang-ge-ce-le)可供参考。EVMGridBot 的使用方式与 TP Bot 基本相同。

- 🐛 Bug 报告：[Issues](https://github.com/fachebot/evm-grid-bot/issues)
- 📧 使用交流：[电报群](https://t.me/+sRrZC-LVPAsyOWE1)

## 🎯 能解决什么问题

### 🔐 私钥安全 - 非托管式设计

**常见工具**：私钥保存在第三方服务器上，存在被窃取或滥用的风险

**EVMGridBot**：私钥始终保存在你的本地设备，永远不会离开你的控制范围

```
┌─────────────────────────────────────────────────────────────┐
│  第三方托管型工具                EVMGridBot                   │
├─────────────────────────────────────────────────────────────┤
│  私钥 → 服务器                    私钥 → 本地仅存储           │
│  服务器可能被攻击                 本地设备完全控制             │
│  信任第三方                       非托管式钱包                 │
└─────────────────────────────────────────────────────────────┘
```

### 🤖 自动化交易 - 解放双手

- **低买高卖**：设置价格区间后，机器人自动在低价买入、高价卖出
- **无需盯盘**：24/7 自动执行策略，告别熬夜盯盘的痛苦
- **任何代币**：支持 BSC 链上任意代币交易，包括 Meme 币、迷因币

### 📱 Telegram 控制 - 极简交互

- 随时随地通过 Telegram 查看策略状态
- 一条指令即可启动快速交易 (`/start quick <token_address>`)
- 实时推送交易通知和盈亏报告

## ✨ 功能特性

- 🚀 **图形化启动器**：提供友好的图形界面，无需命令行操作即可配置和启动
- ⚙️ **配置管理**：可视化配置 RPC URL、OKX API、Telegram Bot，支持配置验证
- 📦 **自动更新**：启动器自动检测并下载最新版本
- 📋 **日志查看**：实时查看程序运行日志，支持级别过滤
- 🔗 BSC 链支持：专为币安智能链优化的交易体验
- 🎯 智能网格交易：在用户设定的价格区间内自动执行低买高卖策略
- 🔗 稳定币交易：使用 USDT（可配置）交易代币，避免主币波动风险
- 📱 Telegram 集成：通过 Telegram Bot 提供便捷的用户交互界面
- 📊 实时监控：通过 Telegram Bot 实时查询盈亏情况和历史交易
- 🛡️ **非托管式安全**：私钥仅存储在本地，不上传任何服务器
- 🔒 **白名单访问控制**：仅白名单用户可操作你的机器人
- 💻 易于部署：支持部署在笔记本、家庭电脑、服务器等环境

## 🏗️ 技术架构

- 开发语言：Go 1.24.1
- 数据库：SQLite3
- 消息接口：Telegram Bot API
- 价格数据：GMGN API/OKX Web3 API
- 架构模式：无外部依赖的单体应用

## 🚀 快速开始

### 💡 推荐方式：使用启动器（适合普通用户）

> 📌 **强烈推荐**：对于普通用户，我们强烈建议使用图形化启动器，它提供了友好的界面和自动配置功能，无需手动编辑配置文件。

**1. 下载启动器**

前往 [Release 页面](https://github.com/fachebot/evm-grid-bot/releases) 下载对应系统架构的启动器压缩包（仅支持 Windows）：
- **Windows 64位**：`launcher-v*-windows-amd64.zip`
- **Windows 32位**：`launcher-v*-windows-386.zip`

**2. 解压并运行**

```bash
# Windows
# 解压 launcher-v*-windows-amd64.zip 后，双击 launcher.exe
```

**3. 配置并启动**

1. 启动器会自动检测并下载最新版本的机器人程序（如果不存在）
2. 在启动器界面中配置以下核心配置项：
   - **RPC URL**：BSC 链的 RPC 地址（点击"修改"按钮进行配置和验证）
   - **OKX API**：OKX Web3 API 密钥（点击"修改"按钮进行配置和验证）
   - **Telegram Bot**：Telegram 机器人 Token（点击"修改"按钮进行配置和验证）
3. 配置验证成功后，点击"启动"按钮运行机器人
4. 在"运行日志"区域查看程序运行状态

> 💡 **提示**：启动器提供了详细的配置引导，包括如何获取 RPC URL、OKX API 密钥和 Telegram Bot Token 的完整步骤。

### 🔧 直接运行程序（适合开发者）

如果您是开发者或需要从源码编译，可以按照以下步骤操作：

**环境要求**

- Git
- Go 1.24.1 或更高版本

**安装部署**

**1. 克隆项目**

```bash
git clone https://github.com/fachebot/evm-grid-bot.git
```

**2. 安装依赖**

```bash
go mod tidy
```

**3. 构建项目**

```bash
go build
```

**4. 运行项目**

```bash
# linux
./evm-grid-bot -f etc/config.yaml

# windows
./evm-grid-bot.exe -f etc/config.yaml
```

> ⚠️ 重要：运行项目前需要创建配置文件，请查看下面的配置说明。

### ⚙️ 配置说明

#### 使用启动器配置（推荐）

启动器提供了图形化配置界面，您可以通过以下方式配置：

1. **RPC URL 配置**：
   - 点击 RPC URL 配置项后的"修改"按钮
   - 输入 BSC RPC 地址
   - 点击"验证"按钮验证 RPC 地址有效性
   - 验证成功后显示链 ID
   - 点击"保存"保存配置

2. **OKX API 配置**：
   - 点击 OKX API 配置项后的"修改"按钮
   - 输入 API Key、Secret Key 和 Passphrase
   - 点击"验证"按钮验证 API 密钥
   - 验证成功后点击"保存"保存配置
   - ⚠️ **重要**：创建 API 密钥前，请确保已在 OKX 开发者平台绑定邮箱和手机号码

3. **Telegram Bot 配置**：
   - 点击 Telegram Bot 配置项后的"修改"按钮
   - 按照对话框中的详细步骤创建 Bot 并获取 Token
   - 输入 Bot Token
   - 点击"验证"按钮验证 Token
   - 验证成功后显示 Bot 用户名
   - 点击"保存"保存配置

> 💡 **提示**：启动器会自动将配置保存到 `etc/config.yaml` 文件中，您也可以点击"📝 打开配置"按钮手动编辑配置文件。

#### 手动配置（直接运行程序）

在运行项目前，需要创建配置文件 `etc/config.yaml`，你可以复制 [etc/config.yaml.sample](etc/config.yaml.sample) 文件到 `etc/config.yaml` 并进行修改：

```yaml
# 链配置
Chain:
  # 链ID
  Id: 56
  # RPC地址
  RpcUrl: "https://1rpc.io/bnb"
  # 原生代币配置
  NativeCurrency:
    Symbol: BNB
    Decimals: 18
  # 稳定币合约地址
  StablecoinCA: "0x55d398326f99059fF775485246999027B3197955"
  # 交易滑点Bps
  SlippageBps: 250
  # DEX聚合器(relay)
  DexAggregator: relay

# 数据API(gmgn/okx)
Datapi: okx

# Okx配置
OkxWeb3:
  Apikey:
  Secretkey:
  Passphrase:

# 代理服务器配置
Sock5Proxy:
  Host: 127.0.0.1 # 代理服务器地址
  Port: 10808 # 代理服务器端口
  Enable: false # 是否启用代理

# 电报机器人配置
TelegramBot:
  Debug: true
  ApiToken: 7916072799:AAFb-C25RgEAxNClxqeRpTkmO6C8e7FhzLs
  WhiteList: # 白名单列表，填写Telegram UserId(非白名单用户不允许使用机器人，如果白名单为空则所有人都可以使用)
    - 993021715

# 默认网格设置
DefaultGridSettings:
  OrderSize: 30 # 每格大小
  MaxGridLimit: 10 # 最大网格数量
  StopLossExit: 0 # 止损金额阈值
  TakeProfitExit: 80 # 盈利目标金额
  TakeProfitRatio: 6 # 止盈百分比(%)
  EnableAutoExit: false # 跌破自动清仓
  LastKlineVolume: 1000 # 最近交易量
  FiveKlineVolume: 0 # 最近5分钟交易量
  GlobalTakeProfitRatio: 0 # 全局止盈涨幅(%)
  DropOn: false # 防瀑布开关
  CandlesToCheck: 3 # 防瀑布K线根数
  DropThreshold: 20 # 防瀑布跌幅阈值百分比(%)

# 快速启动网格设置
QuickStartSettings:
  OrderSize: 30 # 每格大小
  MaxGridLimit: 10 # 最大网格数量
  StopLossExit: 0 # 止损金额阈值
  TakeProfitExit: 80 # 盈利目标金额
  TakeProfitRatio: 6 # 止盈百分比(%)
  EnableAutoExit: false # 跌破自动清仓
  LastKlineVolume: 1000 # 最近交易量
  FiveKlineVolume: 0 # 最近5分钟交易量
  UpperPriceBound: 0.0002 # 网格价格上限
  LowerPriceBound: 0.00005 # 网格价格下限
  GlobalTakeProfitRatio: 0 # 全局止盈涨幅(%)
  DropOn: false # 防瀑布开关
  CandlesToCheck: 3 # 防瀑布K线根数
  DropThreshold: 20 # 防瀑布跌幅阈值百分比(%)

# 创建代币策略的必要条件
TokenRequirements:
  MinMarketCap: 200000 # 最小代币市值
  MinHolderCount: 1500 # 最小代币持有人数
  MinTokenAgeMinutes: 240 # 最小代币年龄(分钟)
  MaxTokenAgeMinutes: 960 # 最高代币年龄(分钟)
```

除了 API 密钥需要使用自己的配置外，其他配置项可使用默认值。默认使用 USDT 进行交易，如需使用其他稳定币可修改 `Chain.StablecoinCA` 配置。

#### 获取必要的 API 密钥

**1. BSC RPC URL**

- 访问 [QuickNode](https://www.quicknode.com/)
- 注册免费账户
- 创建 BSC 主网节点
- 获取 RPC URL
- 将 RPC URL 填写到配置文件 `Chain.RpcUrl` 项

**2. OKX Web3 API**

- 访问 [OKX Web3 开发者平台](https://web3.okx.com/zh-hans/build/dev-portal)
- 使用钱包登录并且在账号设置界面绑定邮箱和手机
- 前往 API Key 页面创建一个新的 API Key
- 将 API Key、Secret Key 和 Passphrase 填写到配置文件 `OkxWeb3` 项

**3. Telegram Bot Token**

- 在 Telegram 中找到 [@BotFather](https://t.me/botfather)
- 发送 /newbot 创建新机器人
- 按提示设置机器人名称和用户名
- 获取 Bot Token
- 将 Bot Token 填写到配置文件 `TelegramBot.ApiToken` 项

#### 网络代理配置

由于网络限制可能导致无法访问 Telegram Bot 服务器，用户可以配置 `Sock5Proxy` 代理来解决连接问题：

```yaml
Sock5Proxy:
  Host: 127.0.0.1
  Port: 10808
  Enable: true # 设置为 true 启用代理
```

## 📖 启动器功能说明

启动器（Launcher）是一个图形化界面工具，提供了以下功能：

- **程序管理**：一键启动和停止机器人程序
- **配置管理**：图形化界面配置核心配置项，支持实时验证
- **版本管理**：自动检测最新版本，支持自动下载
- **日志查看**：实时查看程序运行日志，支持级别过滤和自动滚动
- **配置验证**：自动验证 RPC URL、OKX API 和 Telegram Bot 配置的有效性
- **配置引导**：提供详细的配置步骤说明，帮助用户快速上手

更多启动器使用说明，请查看 [启动器 README](cmd/launcher/README.md)。

## ⚠️ 重要注意事项

### 安全风险

- 🔐 私钥安全：请确保私钥安全，建议使用专门的交易钱包
- 💾 数据备份：运行机器人后会自动创建 `data` 文件夹存储机器人数据，删除此文件夹将丢失所有数据，包括私钥，请谨慎操作

### 交易风险

- 💸 网络费用：每次交易都会产生 BSC 网络手续费
- 📈 市场风险：网格交易适合震荡行情，单边行情可能产生损失
- ⏰ 交易延迟：由于使用免费 API 服务，交易可能存在延迟，不适用于高波动代币交易

### 技术限制

- 🔄 API 限制：注意各 API 服务的调用频率限制
- 🛡️ 反爬虫机制：价格数据从 GMGN 和 OKX 抓取，如因反爬虫机制导致价格获取失败，可尝试修改配置文件 `Datapi` 选项

## 📄 免责声明

**本项目仅供学习和研究使用，使用者需自行承担交易风险。请谨慎使用，作者不对任何损失负责。**

在使用本软件进行交易前，请确保您：

- 充分理解网格交易的风险和机制
- 具备相应的技术知识和风险承受能力
- 仅使用您可以承受损失的资金进行交易

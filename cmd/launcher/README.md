# EVM网格启动器

EVM网格启动器是一个图形化界面工具，用于管理和启动 EVM Grid Bot 程序。提供配置管理、版本更新、日志查看等功能。

## 📋 功能特性

- 🚀 **程序管理**：一键启动和停止 EVM Grid Bot 程序
- ⚙️ **配置管理**：图形化界面配置 RPC URL、OKX API、Telegram Bot
- ✅ **配置验证**：自动验证配置项的有效性
- 📦 **版本管理**：自动检测和下载最新版本
- 📋 **日志查看**：实时查看程序运行日志，支持级别过滤和自动滚动
- 🔄 **自动更新**：可执行文件不存在时自动下载最新版本

## 🏗️ 项目结构

```
cmd/launcher/
├── main.go              # 程序入口
├── main_ui.go           # 主界面实现
├── ui.go                # 初始配置向导界面（可选）
├── configurator.go      # 配置管理器
├── validator.go         # 配置验证器
├── downloader.go        # GitHub Release 下载器
├── process_manager.go   # 进程管理器
└── log_viewer.go        # 日志查看器
```

## 🚀 编译和运行

### 编译

```bash
go build ./cmd/launcher
```

### 运行

```bash
# Windows
.\launcher.exe

# Linux
./launcher
```

## 📖 使用说明

### 启动程序

1. 运行启动器后，如果检测不到可执行文件，会自动下载最新版本
2. 配置核心配置项（RPC URL、OKX API、Telegram Bot）
3. 点击"启动"按钮启动程序

### 配置管理

#### RPC URL 配置

1. 点击 RPC URL 配置项后的"修改"按钮
2. 在对话框中输入 BSC RPC 地址
3. 点击"验证"按钮验证 RPC 地址
4. 验证成功后点击"保存"

#### OKX API 配置

1. 点击 OKX API 配置项后的"修改"按钮
2. 在对话框中输入 API Key、Secret Key 和 Passphrase
3. 点击"验证"按钮验证 API 密钥
4. 验证成功后点击"保存"

**重要提醒**：在创建 API 密钥之前，请确保您的开发者平台账户已经绑定邮箱和手机号码。

#### Telegram Bot 配置

1. 点击 Telegram Bot 配置项后的"修改"按钮
2. 按照对话框中的步骤创建 Bot 并获取 Token
3. 在对话框中输入 Bot Token
4. 点击"验证"按钮验证 Token
5. 验证成功后点击"保存"

### 日志查看

- **级别过滤**：选择日志级别（ALL、INFO、WARN、ERROR、DEBUG）
- **自动滚动**：勾选"自动滚动"后，日志会自动滚动到最新内容
- **清空日志**：点击"清空"按钮清空当前显示的日志

### 版本管理

- **当前版本**：显示当前安装的版本
- **最新版本**：显示 GitHub Release 中的最新版本
- **检查更新**：点击"检查更新"按钮手动检查最新版本
- **自动下载**：如果可执行文件不存在，程序会自动下载最新版本

## 🔧 技术实现

### 依赖

- [Fyne](https://fyne.io/) - 跨平台 GUI 框架
- Go 1.24.1+

### 核心模块

- **Configurator**：管理配置文件（读取、保存、更新）
- **Validator**：验证配置项（RPC、OKX API、Telegram Bot）
- **Downloader**：从 GitHub Release 下载最新版本
- **ProcessManager**：管理 EVM Grid Bot 进程的生命周期
- **LogViewer**：查看和管理程序日志

## 📝 注意事项

1. **配置文件位置**：配置文件位于 `etc/config.yaml`
2. **日志文件位置**：日志文件位于 `logs/gridbot.log`
3. **可执行文件**：程序会在当前目录、`bin/`、`build/` 目录中查找可执行文件
4. **自动下载**：如果找不到可执行文件，程序会自动从 GitHub Release 下载最新版本

## 🐛 问题反馈

如遇到问题，请通过以下方式反馈：

- [GitHub Issues](https://github.com/fachebot/evm-grid-bot/issues)
- [Telegram 群组](https://t.me/+sRrZC-LVPAsyOWE1)

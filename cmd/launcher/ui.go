package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/fachebot/evm-grid-bot/internal/config"
)

// DeployUI 启动器UI（用于初始配置）
type DeployUI struct {
	app          fyne.App
	window       fyne.Window
	currentStep  int
	totalSteps   int
	validator    *Validator
	downloader   *Downloader
	configurator *Configurator
	deployDir    string
	exePath      string

	// 配置项
	rpcUrl            *widget.Entry
	okxApikey         *widget.Entry
	okxSecretkey      *widget.Entry
	okxPassphrase     *widget.Entry
	telegramToken     *widget.Entry
	telegramWhitelist *widget.Entry

	// 验证状态
	rpcValid      bool
	okxValid      bool
	telegramValid bool
}

// NewDeployUI 创建部署UI
func NewDeployUI() *DeployUI {
	myApp := app.NewWithID("evm-grid-bot-deploy")

	window := myApp.NewWindow("EVM Grid Bot 启动器")
	window.Resize(fyne.NewSize(800, 600))
	window.CenterOnScreen()

	return &DeployUI{
		app:         myApp,
		window:      window,
		currentStep: 0,
		totalSteps:  5,
		validator:   &Validator{},
		downloader:  NewDownloader(),
		deployDir:   ".",
	}
}

// Show 显示UI
func (ui *DeployUI) Show() {
	ui.showWelcome()
	ui.window.ShowAndRun()
}

// showWelcome 显示欢迎界面
func (ui *DeployUI) showWelcome() {
	title := widget.NewRichTextFromMarkdown("# EVM Grid Bot 启动器")
	title.Truncation = fyne.TextTruncateEllipsis

	description := widget.NewRichTextFromMarkdown(`
欢迎使用 EVM Grid Bot 启动器！

本工具将帮助您：
1. 自动下载最新版本
2. 配置必要的API密钥
3. 验证配置有效性
4. 启动机器人程序

点击"开始"按钮继续。
	`)

	startBtn := widget.NewButton("开始部署", func() {
		ui.currentStep = 1
		ui.showStep1Download()
	})

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		description,
		widget.NewSeparator(),
		startBtn,
	)

	ui.window.SetContent(container.NewScroll(content))
}

// showStep1Download 显示步骤1：下载
func (ui *DeployUI) showStep1Download() {
	title := widget.NewRichTextFromMarkdown("## 步骤 1/5: 下载最新版本")

	statusLabel := widget.NewLabel("点击开始下载按钮获取最新版本")
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	downloadBtn := widget.NewButton("开始下载", nil)
	nextBtn := widget.NewButton("下一步", func() {
		ui.currentStep = 2
		ui.showStep2RPC()
	})
	nextBtn.Hide()
	nextBtn.Disable()

	downloadBtn.OnTapped = func() {
		downloadBtn.Disable()
		statusLabel.SetText("正在下载...")
		progressBar.Show()
		progressBar.SetValue(0)

		go func() {
			ctx := context.Background()
			// 获取最新release版本
			release, err := ui.downloader.GetLatestRelease(ctx)
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("获取Release信息失败: %w", err), ui.window)
					downloadBtn.Enable()
				})
				return
			}

			// 显示release信息
			fyne.Do(func() {
				statusLabel.SetText(fmt.Sprintf("找到最新版本: %s (Tag: %s)", release.TagName, release.TagName))
			})

			asset, err := ui.downloader.GetAssetForCurrentPlatform(release)
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("未找到适合的文件: %w", err), ui.window)
					downloadBtn.Enable()
				})
				return
			}

			fyne.Do(func() {
				statusLabel.SetText(fmt.Sprintf("正在下载 %s (Tag: %s, %s)...", asset.Name, release.TagName, formatSize(asset.Size)))
			})

			archivePath, err := ui.downloader.DownloadAsset(ctx, asset, ui.deployDir, func(current, total int64) {
				progress := float64(current) / float64(total)
				fyne.Do(func() {
					progressBar.SetValue(progress)
				})
			})
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("下载失败: %w", err), ui.window)
					downloadBtn.Enable()
				})
				return
			}

			fyne.Do(func() {
				statusLabel.SetText("正在解压文件...")
			})
			exePath, err := ui.downloader.ExtractFile(archivePath, ui.deployDir)
			if err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("解压失败: %w", err), ui.window)
					downloadBtn.Enable()
				})
				return
			}

			ui.exePath = exePath
			fyne.Do(func() {
				statusLabel.SetText("下载完成！")
				progressBar.SetValue(1.0)
			})

			// 复制配置文件
			ui.configurator = NewConfigurator(ui.deployDir)
			if err := ui.configurator.CopySampleConfig(); err != nil {
				fyne.Do(func() {
					dialog.ShowError(fmt.Errorf("复制配置文件失败: %w", err), ui.window)
					downloadBtn.Enable()
				})
				return
			}

			// 显示下一步按钮
			fyne.Do(func() {
				downloadBtn.Hide()
				nextBtn.Show()
				nextBtn.Enable()
			})
		}()
	}

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		statusLabel,
		progressBar,
		downloadBtn,
		nextBtn,
	)

	ui.window.SetContent(container.NewScroll(content))
}

// showStep2RPC 显示步骤2：配置RPC
func (ui *DeployUI) showStep2RPC() {
	title := widget.NewRichTextFromMarkdown("## 步骤 2/5: 配置 BSC RPC 地址")

	description := widget.NewRichTextFromMarkdown(`
需要BSC链的RPC地址，您可以选择以下任一服务商注册获取：
	`)

	alchemyBtn := widget.NewButton("注册 Alchemy", func() {
		OpenURL("https://www.alchemy.com/")
	})

	quicknodeBtn := widget.NewButton("注册 QuickNode", func() {
		OpenURL("https://www.quicknode.com/")
	})

	ui.rpcUrl = widget.NewEntry()
	ui.rpcUrl.SetPlaceHolder("请输入BSC RPC地址，例如: https://...")
	// RPC输入框变化时不自动验证，需要用户点击验证按钮

	validateStatus := widget.NewLabel("")
	validateStatus.Hide()

	nextBtn := widget.NewButton("下一步", func() {
		if !ui.rpcValid {
			dialog.ShowError(fmt.Errorf("请先验证RPC地址"), ui.window)
			return
		}
		ui.currentStep = 3
		ui.showStep3OKX()
	})
	nextBtn.Disable()

	validateBtn := widget.NewButton("验证", func() {
		validateStatus.Show()
		validateStatus.SetText("验证中...")
		go func() {
			ui.validateRPCWithStatus(ui.rpcUrl.Text, validateStatus, nextBtn)
		}()
	})

	ui.rpcValid = false

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		description,
		container.NewHBox(alchemyBtn, quicknodeBtn),
		widget.NewSeparator(),
		widget.NewLabel("RPC地址:"),
		ui.rpcUrl,
		container.NewHBox(validateBtn, validateStatus),
		widget.NewSeparator(),
		container.NewHBox(
			widget.NewButton("上一步", func() {
				ui.currentStep = 1
				ui.showStep1Download()
			}),
			nextBtn,
		),
	)

	ui.window.SetContent(container.NewScroll(content))
}

// validateRPCWithStatus 验证RPC地址并更新状态
func (ui *DeployUI) validateRPCWithStatus(rpcUrl string, statusLabel *widget.Label, nextBtn *widget.Button) {
	if rpcUrl == "" {
		fyne.Do(func() {
			statusLabel.SetText("请输入RPC地址")
			statusLabel.Importance = widget.WarningImportance
		})
		return
	}

	err := ui.validator.ValidateRPC(rpcUrl, 56)
	if err != nil {
		fyne.Do(func() {
			statusLabel.SetText("✗ 验证失败: " + err.Error())
			statusLabel.Importance = widget.DangerImportance
			ui.rpcValid = false
			nextBtn.Disable()
		})
	} else {
		// 保存配置
		ui.configurator.UpdateConfig(func(cfg *config.Config) {
			cfg.Chain.RpcUrl = rpcUrl
		})

		fyne.Do(func() {
			statusLabel.SetText("✓ 验证成功")
			statusLabel.Importance = widget.SuccessImportance
			ui.rpcValid = true
			nextBtn.Enable()
		})
	}
}

// showStep3OKX 显示步骤3：配置OKX
func (ui *DeployUI) showStep3OKX() {
	title := widget.NewRichTextFromMarkdown("## 步骤 3/5: 配置 OKX Web3 API")

	description := widget.NewRichTextFromMarkdown(`
访问 [OKX Web3 开发者平台](https://web3.okx.com/zh-hans/build/dev-portal) 注册并创建API密钥
	`)

	okxLinkBtn := widget.NewButton("打开 OKX Web3 开发者平台", func() {
		OpenURL("https://web3.okx.com/zh-hans/build/dev-portal")
	})

	ui.okxApikey = widget.NewEntry()
	ui.okxApikey.SetPlaceHolder("API Key")

	ui.okxSecretkey = widget.NewPasswordEntry()
	ui.okxSecretkey.SetPlaceHolder("Secret Key")

	ui.okxPassphrase = widget.NewPasswordEntry()
	ui.okxPassphrase.SetPlaceHolder("Passphrase")

	validateStatus := widget.NewLabel("")
	validateStatus.Hide()

	nextBtn := widget.NewButton("下一步", func() {
		if !ui.okxValid {
			dialog.ShowError(fmt.Errorf("请先验证OKX API密钥"), ui.window)
			return
		}
		ui.currentStep = 4
		ui.showStep4Telegram()
	})
	nextBtn.Disable()

	validateBtn := widget.NewButton("验证", func() {
		validateStatus.Show()
		validateStatus.SetText("验证中...")
		go func() {
			ui.validateOKXWithStatus(validateStatus, nextBtn)
		}()
	})

	ui.okxValid = false

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		description,
		okxLinkBtn,
		widget.NewSeparator(),
		widget.NewLabel("API Key:"),
		ui.okxApikey,
		widget.NewLabel("Secret Key:"),
		ui.okxSecretkey,
		widget.NewLabel("Passphrase:"),
		ui.okxPassphrase,
		container.NewHBox(validateBtn, validateStatus),
		widget.NewSeparator(),
		container.NewHBox(
			widget.NewButton("上一步", func() {
				ui.currentStep = 2
				ui.showStep2RPC()
			}),
			nextBtn,
		),
	)

	ui.window.SetContent(container.NewScroll(content))
}

// validateOKXWithStatus 验证OKX API并更新状态
func (ui *DeployUI) validateOKXWithStatus(statusLabel *widget.Label, nextBtn *widget.Button) {
	apikey := ui.okxApikey.Text
	secretkey := ui.okxSecretkey.Text
	passphrase := ui.okxPassphrase.Text

	if apikey == "" || secretkey == "" || passphrase == "" {
		fyne.Do(func() {
			statusLabel.SetText("✗ 请填写完整的OKX API信息")
			statusLabel.Importance = widget.WarningImportance
			ui.okxValid = false
			nextBtn.Disable()
		})
		return
	}

	err := ui.validator.ValidateOKX(apikey, secretkey, passphrase)
	if err != nil {
		fyne.Do(func() {
			statusLabel.SetText("✗ 验证失败: " + err.Error())
			statusLabel.Importance = widget.DangerImportance
			ui.okxValid = false
			nextBtn.Disable()
		})
	} else {
		// 保存配置
		ui.configurator.UpdateConfig(func(cfg *config.Config) {
			cfg.OkxWeb3.Apikey = apikey
			cfg.OkxWeb3.Secretkey = secretkey
			cfg.OkxWeb3.Passphrase = passphrase
		})

		fyne.Do(func() {
			statusLabel.SetText("✓ 验证成功")
			statusLabel.Importance = widget.SuccessImportance
			ui.okxValid = true
			nextBtn.Enable()
		})
	}
}

// showStep4Telegram 显示步骤4：配置Telegram
func (ui *DeployUI) showStep4Telegram() {
	title := widget.NewRichTextFromMarkdown("## 步骤 4/5: 配置 Telegram Bot")

	description := widget.NewRichTextFromMarkdown(`
访问 [@BotFather](https://t.me/botfather) 创建机器人并获取Token
	`)

	telegramLinkBtn := widget.NewButton("打开 Telegram BotFather", func() {
		OpenURL("https://t.me/botfather")
	})

	ui.telegramToken = widget.NewPasswordEntry()
	ui.telegramToken.SetPlaceHolder("Bot Token")

	ui.telegramWhitelist = widget.NewEntry()
	ui.telegramWhitelist.SetPlaceHolder("白名单用户ID（可选，多个用逗号分隔，留空表示所有人可用）")

	validateStatus := widget.NewLabel("")
	validateStatus.Hide()

	nextBtn := widget.NewButton("下一步", func() {
		if !ui.telegramValid {
			dialog.ShowError(fmt.Errorf("请先验证Telegram Bot Token"), ui.window)
			return
		}
		ui.currentStep = 5
		ui.showStep5Complete()
	})
	nextBtn.Disable()

	validateBtn := widget.NewButton("验证", func() {
		validateStatus.Show()
		validateStatus.SetText("验证中...")
		go func() {
			ui.validateTelegramWithStatus(validateStatus, nextBtn)
		}()
	})

	ui.telegramValid = false

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		description,
		telegramLinkBtn,
		widget.NewSeparator(),
		widget.NewLabel("Bot Token:"),
		ui.telegramToken,
		widget.NewLabel("白名单（可选）:"),
		ui.telegramWhitelist,
		container.NewHBox(validateBtn, validateStatus),
		widget.NewSeparator(),
		container.NewHBox(
			widget.NewButton("上一步", func() {
				ui.currentStep = 3
				ui.showStep3OKX()
			}),
			nextBtn,
		),
	)

	ui.window.SetContent(container.NewScroll(content))
}

// validateTelegramWithStatus 验证Telegram Token并更新状态
func (ui *DeployUI) validateTelegramWithStatus(statusLabel *widget.Label, nextBtn *widget.Button) {
	token := ui.telegramToken.Text
	if token == "" {
		fyne.Do(func() {
			statusLabel.SetText("✗ 请填写Telegram Bot Token")
			statusLabel.Importance = widget.WarningImportance
			ui.telegramValid = false
			nextBtn.Disable()
		})
		return
	}

	err := ui.validator.ValidateTelegram(token)
	if err != nil {
		fyne.Do(func() {
			statusLabel.SetText("✗ 验证失败: " + err.Error())
			statusLabel.Importance = widget.DangerImportance
			ui.telegramValid = false
			nextBtn.Disable()
		})
	} else {
		// 解析白名单
		var whitelist []int64
		whitelistStr := strings.TrimSpace(ui.telegramWhitelist.Text)
		if whitelistStr != "" {
			parts := strings.Split(whitelistStr, ",")
			for _, part := range parts {
				id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
				if err == nil {
					whitelist = append(whitelist, id)
				}
			}
		}

		// 保存配置
		ui.configurator.UpdateConfig(func(cfg *config.Config) {
			cfg.TelegramBot.ApiToken = token
			cfg.TelegramBot.WhiteList = whitelist
		})

		fyne.Do(func() {
			statusLabel.SetText("✓ 验证成功")
			statusLabel.Importance = widget.SuccessImportance
			ui.telegramValid = true
			nextBtn.Enable()
		})
	}
}

// showStep5Complete 显示步骤5：完成
func (ui *DeployUI) showStep5Complete() {
	title := widget.NewRichTextFromMarkdown("## 步骤 5/5: 部署完成")

	summary := widget.NewRichTextFromMarkdown(`
配置已完成！点击"启动程序"按钮开始运行机器人。

**配置摘要：**
- RPC地址: 已配置
- OKX API: 已配置
- Telegram Bot: 已配置
	`)

	startBtn := widget.NewButton("启动程序", func() {
		ui.startProgram()
	})

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		summary,
		widget.NewSeparator(),
		startBtn,
		container.NewHBox(
			widget.NewButton("上一步", func() {
				ui.currentStep = 4
				ui.showStep4Telegram()
			}),
		),
	)

	ui.window.SetContent(container.NewScroll(content))
}

// startProgram 启动程序
func (ui *DeployUI) startProgram() {
	if ui.exePath == "" {
		dialog.ShowError(fmt.Errorf("可执行文件路径未找到"), ui.window)
		return
	}

	// 切换到部署目录
	workDir, _ := filepath.Abs(ui.deployDir)

	cmd := exec.Command(ui.exePath)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		dialog.ShowError(fmt.Errorf("启动程序失败: %w", err), ui.window)
		return
	}

	dialog.ShowInformation("启动成功", "程序已在后台启动！", ui.window)
	ui.window.Close()
}

// formatSize 格式化文件大小
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

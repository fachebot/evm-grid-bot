package main

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2/app"
)

func main() {
	// 获取当前工作目录
	deployDir, err := os.Getwd()
	if err != nil {
		deployDir = "."
	}

	// 确保etc目录存在
	etcDir := filepath.Join(deployDir, "etc")
	if _, err := os.Stat(etcDir); os.IsNotExist(err) {
		os.MkdirAll(etcDir, 0755)
	}

	// 创建应用
	myApp := app.NewWithID("evm-grid-bot-manager")
	
	// 创建主界面
	mainUI := NewMainUI(myApp, deployDir)
	mainUI.Show()
}

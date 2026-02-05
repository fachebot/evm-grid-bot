package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ProcessManager 进程管理器
type ProcessManager struct {
	cmd        *exec.Cmd
	isRunning  bool
	logBuffer  *bytes.Buffer
	logLines   []string
	maxLogLines int
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	onLogLine  func(level, message string) // 日志回调函数
}

// NewProcessManager 创建进程管理器
func NewProcessManager() *ProcessManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ProcessManager{
		logBuffer:   &bytes.Buffer{},
		logLines:    make([]string, 0),
		maxLogLines: 1000, // 最多保留1000行日志
		ctx:         ctx,
		cancel:      cancel,
	}
}

// SetLogCallback 设置日志回调函数
func (pm *ProcessManager) SetLogCallback(callback func(level, message string)) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.onLogLine = callback
}

// Start 启动程序并捕获日志
func (pm *ProcessManager) Start(exePath, workDir string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.isRunning {
		return fmt.Errorf("程序已在运行中")
	}

	// 检查可执行文件是否存在
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return fmt.Errorf("可执行文件不存在: %s", exePath)
	}

	// 创建工作目录
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("获取工作目录失败: %w", err)
	}

	// 如果 context 已被取消，创建新的 context
	select {
	case <-pm.ctx.Done():
		// context 已被取消，创建新的
		pm.ctx, pm.cancel = context.WithCancel(context.Background())
	default:
		// context 仍然有效，继续使用
	}

	// 创建命令
	pm.cmd = exec.CommandContext(pm.ctx, exePath)
	pm.cmd.Dir = absWorkDir
	pm.cmd.Env = os.Environ()

	// 创建管道捕获stdout和stderr
	stdoutPipe, err := pm.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建stdout管道失败: %w", err)
	}

	stderrPipe, err := pm.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建stderr管道失败: %w", err)
	}

	// 启动程序
	if err := pm.cmd.Start(); err != nil {
		return fmt.Errorf("启动程序失败: %w", err)
	}

	pm.isRunning = true

	// 启动goroutine读取stdout
	go pm.readLogs(stdoutPipe, "INFO")
	
	// 启动goroutine读取stderr
	go pm.readLogs(stderrPipe, "ERROR")

	// 启动goroutine等待进程结束
	go pm.waitForProcess()

	return nil
}

// readLogs 读取日志输出
func (pm *ProcessManager) readLogs(pipe io.ReadCloser, defaultLevel string) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		
		pm.mu.Lock()
		// 添加到缓冲区
		pm.logBuffer.WriteString(line)
		pm.logBuffer.WriteString("\n")
		
		// 添加到日志行列表
		pm.logLines = append(pm.logLines, line)
		
		// 限制日志行数
		if len(pm.logLines) > pm.maxLogLines {
			pm.logLines = pm.logLines[len(pm.logLines)-pm.maxLogLines:]
		}
		
		// 解析日志级别
		level := pm.parseLogLevel(line, defaultLevel)
		message := line
		
		callback := pm.onLogLine
		pm.mu.Unlock()
		
		// 调用回调函数
		if callback != nil {
			callback(level, message)
		}
	}
}

// parseLogLevel 解析日志级别
func (pm *ProcessManager) parseLogLevel(line, defaultLevel string) string {
	lineUpper := strings.ToUpper(line)
	if strings.Contains(lineUpper, "[ERROR]") || strings.Contains(lineUpper, "ERROR") {
		return "ERROR"
	}
	if strings.Contains(lineUpper, "[WARN]") || strings.Contains(lineUpper, "WARN") {
		return "WARN"
	}
	if strings.Contains(lineUpper, "[INFO]") || strings.Contains(lineUpper, "INFO") {
		return "INFO"
	}
	if strings.Contains(lineUpper, "[DEBUG]") || strings.Contains(lineUpper, "DEBUG") {
		return "DEBUG"
	}
	return defaultLevel
}

// waitForProcess 等待进程结束
func (pm *ProcessManager) waitForProcess() {
	if pm.cmd == nil {
		return
	}
	
	err := pm.cmd.Wait()
	
	pm.mu.Lock()
	pm.isRunning = false
	pm.mu.Unlock()
	
	if err != nil && pm.onLogLine != nil {
		pm.onLogLine("ERROR", fmt.Sprintf("进程异常退出: %v", err))
	} else if pm.onLogLine != nil {
		pm.onLogLine("INFO", "进程已退出")
	}
}

// Stop 停止程序
func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.isRunning || pm.cmd == nil || pm.cmd.Process == nil {
		return fmt.Errorf("程序未运行")
	}

	// 取消上下文
	pm.cancel()

	// 发送SIGTERM信号（优雅停止）
	if runtime.GOOS != "windows" {
		if err := pm.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// 如果SIGTERM失败，尝试SIGKILL
			pm.cmd.Process.Kill()
			return err
		}
		
		// 等待进程退出，最多等待5秒
		done := make(chan error, 1)
		go func() {
			done <- pm.cmd.Wait()
		}()
		
		select {
		case <-time.After(5 * time.Second):
			// 超时，强制kill
			pm.cmd.Process.Kill()
			<-done
		case err := <-done:
			if err != nil {
				pm.cmd.Process.Kill()
			}
		}
	} else {
		// Windows平台
		pm.cmd.Process.Kill()
	}

	pm.isRunning = false
	return nil
}

// IsRunning 检查程序是否运行
func (pm *ProcessManager) IsRunning() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.isRunning
}

// GetLogs 获取所有日志内容
func (pm *ProcessManager) GetLogs() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.logBuffer.String()
}

// GetLogLines 获取日志行列表
func (pm *ProcessManager) GetLogLines() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	result := make([]string, len(pm.logLines))
	copy(result, pm.logLines)
	return result
}

// ClearLogs 清空日志
func (pm *ProcessManager) ClearLogs() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.logBuffer.Reset()
	pm.logLines = pm.logLines[:0]
}

// GetProcessID 获取进程ID
func (pm *ProcessManager) GetProcessID() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pm.cmd != nil && pm.cmd.Process != nil {
		return pm.cmd.Process.Pid
	}
	return 0
}

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// LogViewer 日志查看器
type LogViewer struct {
	logText        *widget.RichText
	logScroll      *container.Scroll // 滚动容器引用
	logLines       []LogLine
	maxLines       int
	logLevelFilter string
	mu             sync.RWMutex
	autoScroll     bool
}

// LogLine 日志行
type LogLine struct {
	Time    string
	Level   string
	Message string
	Raw     string
}

// NewLogViewer 创建日志查看器
func NewLogViewer() *LogViewer {
	return &LogViewer{
		logLines:       make([]LogLine, 0),
		maxLines:       1000,
		logLevelFilter: "ALL",
		autoScroll:     true,
	}
}

// SetLogTextWidget 设置日志显示组件
func (lv *LogViewer) SetLogTextWidget(text *widget.RichText) {
	lv.mu.Lock()
	defer lv.mu.Unlock()
	lv.logText = text
}

// SetLogScrollContainer 设置滚动容器
func (lv *LogViewer) SetLogScrollContainer(scroll *container.Scroll) {
	lv.mu.Lock()
	defer lv.mu.Unlock()
	lv.logScroll = scroll
}

// AppendLog 追加日志到显示区域
func (lv *LogViewer) AppendLog(level, message string) {
	// 解析日志行
	logLine := lv.parseLogLine(message, level)
	
	// 添加到日志列表（需要锁保护）
	lv.mu.Lock()
	lv.logLines = append(lv.logLines, logLine)
	
	// 限制日志行数
	if len(lv.logLines) > lv.maxLines {
		lv.logLines = lv.logLines[len(lv.logLines)-lv.maxLines:]
	}
	lv.mu.Unlock()
	
	// 更新显示（在锁外调用，避免死锁）
	lv.updateDisplay()
}

// parseLogLine 解析日志行
func (lv *LogViewer) parseLogLine(raw, defaultLevel string) LogLine {
	line := LogLine{
		Time:    time.Now().Format("15:04:05"), // 使用更紧凑的时间格式
		Level:   defaultLevel,
		Message: raw,
		Raw:     raw,
	}

	// 尝试解析logrus格式的日志
	// 格式示例: time="2024-01-15T10:30:15+08:00" level=info msg="message"
	// 或者: 2024-01-15 10:30:15 [INFO] message
	
	// 尝试匹配时间戳格式
	timePattern := regexp.MustCompile(`(\d{4}-\d{2}-\d{2}[\sT]\d{2}:\d{2}:\d{2})`)
	if matches := timePattern.FindStringSubmatch(raw); len(matches) > 1 {
		line.Time = matches[1]
	}
	
	// 尝试匹配日志级别
	levelPattern := regexp.MustCompile(`(?i)\[?(INFO|WARN|ERROR|DEBUG|FATAL)\]?`)
	if matches := levelPattern.FindStringSubmatch(raw); len(matches) > 1 {
		line.Level = strings.ToUpper(matches[1])
	}
	
	// 提取消息内容（移除时间戳和级别）
	message := raw
	extractedFromJSON := false
	// 尝试提取JSON格式中的msg字段
	if strings.Contains(message, `"msg"`) {
		msgPattern := regexp.MustCompile(`"msg"\s*:\s*"([^"]*)"`)
		if matches := msgPattern.FindStringSubmatch(message); len(matches) > 1 {
			message = matches[1]
			extractedFromJSON = true
		} else {
			// 如果提取失败，尝试移除JSON格式的其他部分
			msgPattern2 := regexp.MustCompile(`.*"msg"\s*:\s*"([^"]*)"`)
			if matches := msgPattern2.FindStringSubmatch(message); len(matches) > 1 {
				message = matches[1]
				extractedFromJSON = true
			}
		}
	}
	
	// 移除时间戳和级别标记（仅在未从JSON提取时执行，避免误删消息内容中的[]）
	if !extractedFromJSON {
		if idx := strings.Index(message, "]"); idx > 0 {
			message = message[idx+1:]
		}
	}
	// 移除JSON格式的level和time字段
	message = regexp.MustCompile(`"level"\s*:\s*"[^"]*"`).ReplaceAllString(message, "")
	message = regexp.MustCompile(`"time"\s*:\s*"[^"]*"`).ReplaceAllString(message, "")
	message = strings.ReplaceAll(message, "{", "")
	message = strings.ReplaceAll(message, "}", "")
	message = strings.ReplaceAll(message, ",", "")
	
	message = strings.TrimSpace(message)
	// 清理消息中的换行符和多余空格，确保在同一行显示
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\r", " ")
	message = strings.ReplaceAll(message, "\t", " ")
	// 移除多余的空格
	message = regexp.MustCompile(`\s+`).ReplaceAllString(message, " ")
	if message != "" {
		line.Message = message
	}

	return line
}

// updateDisplay 更新显示
func (lv *LogViewer) updateDisplay() {
	// 在后台goroutine中准备数据
	lv.mu.RLock()
	filteredLines := lv.filterLogs()
	autoScroll := lv.autoScroll
	logScroll := lv.logScroll
	lv.mu.RUnlock()
	
	// 准备日志段数据
	segments := make([]widget.RichTextSegment, 0)
	for _, logLine := range filteredLines {
		// 提取时间部分（只显示时分秒，更紧凑）
		timeStr := logLine.Time
		if len(timeStr) > 10 {
			// 如果包含日期，只取时间部分
			parts := strings.Fields(timeStr)
			if len(parts) > 1 {
				timeStr = parts[1] // 取时间部分
			} else if strings.Contains(timeStr, "T") {
				// ISO格式：2024-01-15T10:30:15
				if idx := strings.Index(timeStr, "T"); idx > 0 {
					timeStr = timeStr[idx+1:]
					if len(timeStr) > 8 {
						timeStr = timeStr[:8] // 只取 HH:MM:SS
					}
				}
			}
		}
		
		// 清理消息中的换行符和JSON格式，确保在同一行显示
		message := strings.TrimSpace(logLine.Message)
		// 移除所有换行符和回车符
		message = strings.ReplaceAll(message, "\n", " ")
		message = strings.ReplaceAll(message, "\r", " ")
		// 移除JSON格式中的多余空格和换行
		message = strings.ReplaceAll(message, "  ", " ")
		// 如果消息包含JSON格式，尝试提取msg字段
		if strings.Contains(message, `"msg"`) {
			msgPattern := regexp.MustCompile(`"msg"\s*:\s*"([^"]*)"`)
			if matches := msgPattern.FindStringSubmatch(message); len(matches) > 1 {
				message = matches[1]
			}
		}
		
		// 转义消息中的方括号，避免被UI框架隐藏
		// 将 [ 和 ] 替换为全角字符，确保正常显示
		message = strings.ReplaceAll(message, "[", "［")
		message = strings.ReplaceAll(message, "]", "］")
		
		// 级别颜色
		levelColor := lv.getLevelColor(logLine.Level)
		
		// 将所有内容合并到一个TextSegment中，确保在同一行显示
		// 格式: [时间] [级别] 消息内容
		fullLine := fmt.Sprintf("[%s] [%s] %s\n", timeStr, logLine.Level, message)
		
		// 创建一个完整的文本段，使用级别颜色
		lineSeg := &widget.TextSegment{
			Text: fullLine,
			Style: widget.RichTextStyle{
				ColorName: levelColor, // 使用级别颜色
				TextStyle: fyne.TextStyle{Monospace: true},
			},
		}
		segments = append(segments, lineSeg)
	}
	
	// 使用 fyne.Do 确保在主GUI线程中执行所有GUI操作
	fyne.Do(func() {
		if lv.logText == nil {
			return
		}
		
		// 清空现有内容并设置新内容
		lv.logText.Segments = segments
		
		// 刷新显示
		lv.logText.Refresh()
		
		// 如果启用了自动滚动，滚动到底部
		if autoScroll && logScroll != nil {
			logScroll.ScrollToBottom()
		}
	})
}

// filterLogs 过滤日志（需要在持有锁的情况下调用）
func (lv *LogViewer) filterLogs() []LogLine {
	if lv.logLevelFilter == "ALL" {
		return lv.logLines
	}

	filtered := make([]LogLine, 0)
	for _, line := range lv.logLines {
		if strings.EqualFold(line.Level, lv.logLevelFilter) {
			filtered = append(filtered, line)
		}
	}
	return filtered
}

// getLevelColor 获取日志级别对应的颜色
func (lv *LogViewer) getLevelColor(level string) fyne.ThemeColorName {
	switch strings.ToUpper(level) {
	case "ERROR", "FATAL":
		return theme.ColorNameError
	case "WARN", "WARNING":
		return theme.ColorNameWarning
	case "INFO":
		return theme.ColorNamePrimary
	case "DEBUG":
		return theme.ColorNamePlaceHolder
	default:
		return theme.ColorNameForeground
	}
}

// ClearLogs 清空日志显示
func (lv *LogViewer) ClearLogs() {
	lv.mu.Lock()
	lv.logLines = lv.logLines[:0]
	lv.mu.Unlock()
	
	// 在锁外更新GUI，使用fyne.Do确保在主线程
	fyne.Do(func() {
		if lv.logText != nil {
			lv.logText.Segments = []widget.RichTextSegment{}
			lv.logText.Refresh()
		}
	})
}

// SetLogLevelFilter 设置日志级别过滤
func (lv *LogViewer) SetLogLevelFilter(level string) {
	lv.mu.Lock()
	lv.logLevelFilter = level
	lv.mu.Unlock()
	// 在锁外更新显示，避免死锁
	lv.updateDisplay()
}

// LoadLogsFromFile 从日志文件加载日志
func (lv *LogViewer) LoadLogsFromFile(logFilePath string) error {
	file, err := os.Open(logFilePath)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer file.Close()

	// 读取最后N行（避免加载过多）
	lines := lv.readLastLines(file, lv.maxLines)
	
	// 在锁保护下添加日志
	lv.mu.Lock()
	for _, line := range lines {
		logLine := lv.parseLogLine(line, "INFO")
		lv.logLines = append(lv.logLines, logLine)
	}
	lv.mu.Unlock()
	
	// 在锁外更新显示，避免死锁
	lv.updateDisplay()
	
	return nil
}

// readLastLines 读取文件的最后N行
func (lv *LogViewer) readLastLines(file io.Reader, n int) []string {
	scanner := bufio.NewScanner(file)
	lines := make([]string, 0)
	
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:] // 移除第一行
		}
	}
	
	return lines
}

// WatchLogFile 监控日志文件变化
func (lv *LogViewer) WatchLogFile(logFilePath string, onNewLine func(string)) error {
	// 使用简单的轮询方式监控文件
	// 实际项目中可以使用fsnotify库
	go func() {
		var lastSize int64 = 0
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			info, err := os.Stat(logFilePath)
			if err != nil {
				continue
			}

			if info.Size() > lastSize {
				// 文件有新内容，读取新增部分
				file, err := os.Open(logFilePath)
				if err != nil {
					continue
				}

				// 定位到上次读取的位置
				file.Seek(lastSize, 0)
				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					line := scanner.Text()
					if onNewLine != nil {
						onNewLine(line)
					}
				}
				file.Close()

				lastSize = info.Size()
			}
		}
	}()

	return nil
}

// GetLogCount 获取日志行数
func (lv *LogViewer) GetLogCount() int {
	lv.mu.RLock()
	defer lv.mu.RUnlock()
	return len(lv.logLines)
}

// SetAutoScroll 设置自动滚动
func (lv *LogViewer) SetAutoScroll(auto bool) {
	lv.mu.Lock()
	defer lv.mu.Unlock()
	lv.autoScroll = auto
}

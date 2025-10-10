package logger

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	*logrus.Logger
	fileLogger *logrus.Logger
}

var defaultLogger *Logger

func init() {
	// 控制台日志配置
	consoleLogger := logrus.New()
	consoleLogger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
	consoleLogger.SetOutput(os.Stdout)
	consoleLogger.SetLevel(logrus.DebugLevel)

	// 文件日志配置
	fileLogger := logrus.New()
	fileLogger.SetFormatter(&logrus.JSONFormatter{
		PrettyPrint:     false,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	fileLogger.SetLevel(logrus.InfoLevel)

	// 创建日志目录
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		consoleLogger.Errorf("无法创建日志目录: %v", err)
	}

	// 使用lumberjack进行日志轮转
	logFile := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "gridbot.log"),
		MaxSize:    10,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
	}

	fileLogger.SetOutput(logFile)

	defaultLogger = &Logger{
		Logger:     consoleLogger,
		fileLogger: fileLogger,
	}
}

func Infof(format string, args ...any) {
	defaultLogger.Logger.Infof(format, args...)
	defaultLogger.fileLogger.Infof(format, args...)
}

func Warnf(format string, args ...any) {
	defaultLogger.Logger.Warnf(format, args...)
	defaultLogger.fileLogger.Warnf(format, args...)
}

func Errorf(format string, args ...any) {
	defaultLogger.Logger.Errorf(format, args...)
	defaultLogger.fileLogger.Errorf(format, args...)
}

func Fatalf(format string, args ...any) {
	defaultLogger.Logger.Fatalf(format, args...)
	defaultLogger.fileLogger.Fatalf(format, args...)
}

func Debugf(format string, args ...any) {
	defaultLogger.Logger.Debugf(format, args...)
	defaultLogger.fileLogger.Debugf(format, args...)
}

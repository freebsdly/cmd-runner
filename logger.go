package main

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ParseLevel 将字符串转为 zapcore.Level
func ParseLevel(levelStr string) (zapcore.Level, error) {
	switch strings.ToLower(levelStr) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	case "panic":
		return zapcore.PanicLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown log level: %s", levelStr)
	}
}

// NewLogger 根据配置创建 zap.Logger
func NewLogger(levelStr, format, output string) (*zap.Logger, error) {
	level, err := ParseLevel(levelStr)
	if err != nil {
		return nil, err
	}

	var encoder zapcore.Encoder
	if format == "json" {
		encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	} else {
		// console (human-readable)
		encCfg := zap.NewDevelopmentEncoderConfig()
		encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder // 彩色级别
		encoder = zapcore.NewConsoleEncoder(encCfg)
	}

	var writer zapcore.WriteSyncer
	switch output {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	default:
		// 假设是文件路径
		file, err := os.OpenFile(output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %w", output, err)
		}
		writer = zapcore.AddSync(file)
	}

	core := zapcore.NewCore(encoder, writer, level)
	return zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)), nil
}

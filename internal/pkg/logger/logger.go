package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"devops-cd/internal/pkg/config"
)

var Log *zap.Logger
var log *zap.Logger
var logWriter *LogWriter

// customTimeEncoder 自定义时间格式编码器
// 输出格式: 2006-01-02 15:04:05.000
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// customCallerEncoder 自定义调用者编码器
// 输出相对于项目根目录的完整路径，支持IDE点击跳转
// 格式: internal/pkg/logger/logger.go:45
func customCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	if !caller.Defined {
		enc.AppendString("undefined")
		return
	}

	// 获取调用者的完整路径
	fullPath := caller.File

	// 查找项目根目录标识（go.mod所在目录）
	_, currentFile, _, ok := runtime.Caller(0)
	if ok {
		// 向上查找go.mod
		dir := filepath.Dir(currentFile)
		for i := 0; i < 10; i++ { // 最多向上查找10层
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				// 找到项目根目录，计算相对路径
				if rel, err := filepath.Rel(dir, fullPath); err == nil {
					enc.AppendString(rel + ":" + caller.String()[strings.LastIndex(caller.String(), ":")+1:])
					return
				}
				break
			}
			parentDir := filepath.Dir(dir)
			if parentDir == dir {
				break // 已到达根目录
			}
			dir = parentDir
		}
	}

	// 如果无法找到项目根目录，使用短路径
	enc.AppendString(caller.TrimmedPath())
}

// Init 初始化日志
func Init(cfg *config.LogConfig) error {
	// 设置日志级别
	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// 编码器配置（基础配置）
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:          "time",
		LevelKey:         "level",
		NameKey:          "logger",
		CallerKey:        "caller",
		MessageKey:       "msg",
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeTime:       customTimeEncoder, // 自定义时间格式
		EncodeDuration:   zapcore.SecondsDurationEncoder,
		EncodeCaller:     customCallerEncoder, // 自定义调用者格式，支持点击跳转
		ConsoleSeparator: " ",                 // 字段分隔符
	}

	// 选择编码器
	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		// JSON格式: 纯JSON输出（不使用颜色）
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		// Console格式: 时间 INFO 代码位置 日志消息 {json格式参数}
		// 使用带颜色的级别编码器
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // 彩色级别: INFO, ERROR
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 输出位置
	var writeSyncer zapcore.WriteSyncer
	if cfg.Output == "stdout" || cfg.FilePath == "" {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else {
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		writeSyncer = zapcore.AddSync(file)
	}

	// 创建核心
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// 创建logger
	Log = zap.New(core, zap.AddCaller())
	log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	logWriter = &LogWriter{writeSyncer}

	return nil
}

// Close 关闭日志
func Close() error {
	var err1, err2 error
	if Log != nil {
		err1 = Log.Sync()
	}
	if log != nil {
		err2 = log.Sync()
	}

	if err1 != nil || err2 != nil {
		return fmt.Errorf("close log error: %v, %v", err1, err2)
	}
	return nil
}

// Debug 输出Debug日志
func Debug(msg string, fields ...zap.Field) {
	log.Debug(msg, fields...)
}

// Info 输出Info日志
func Info(msg string, fields ...zap.Field) {
	log.Info(msg, fields...)
}

// Warn 输出Warn日志
func Warn(msg string, fields ...zap.Field) {
	log.Warn(msg, fields...)
}

// Error 输出Error日志
func Error(msg string, fields ...zap.Field) {
	log.Error(msg, fields...)
}

// Fatal 输出Fatal日志
func Fatal(msg string, fields ...zap.Field) {
	log.Fatal(msg, fields...)
}

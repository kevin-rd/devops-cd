package logger

import (
	"fmt"
	"go.uber.org/zap/zapcore"
)

type LogWriter struct {
	zapcore.WriteSyncer
}

func (l *LogWriter) Printf(format string, args ...interface{}) {
	_, _ = l.WriteSyncer.Write([]byte(fmt.Sprintf(format, args...)))
	_ = l.WriteSyncer.Sync()
}

func GetWriter() *LogWriter {
	return logWriter
}

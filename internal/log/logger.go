// Package log 基于 zap 提供日志初始化，支持 console 和 null 两种输出模式。
package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 是全局的 sugared logger 实例。
var Logger *zap.SugaredLogger

// Init 根据 logTo 参数初始化日志系统。
// "console" 输出带颜色的终端日志，"null" 则静默所有日志。
func Init(logTo string) {
	if logTo == "null" {
		Logger = zap.NewNop().Sugar()
		return
	}

	cfg := zap.NewDevelopmentConfig()
	cfg.Encoding = "console"
	cfg.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:     "time",
		LevelKey:    "level",
		NameKey:     "logger",
		CallerKey:   zapcore.OmitKey,
		FunctionKey: zapcore.OmitKey,
		MessageKey:  "msg",
		LineEnding:  zapcore.DefaultLineEnding,
		EncodeLevel: zapcore.CapitalColorLevelEncoder,
		EncodeTime:  zapcore.TimeEncoderOfLayout("15:04:05"),
	}
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	Logger = logger.Sugar()
}

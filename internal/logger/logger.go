package logger

import (
	"github.com/teelyjc/home/internal/constants"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func SetupLogger() {
	config := &zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.DebugLevel),
		Development: constants.IsDevelopment,
		Encoding: func() string {
			if constants.IsDevelopment {
				return "console"
			} else {
				return "json"
			}
		}(),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:    "timestamp",
			LevelKey:   "level",
			NameKey:    "logger",
			CallerKey:  "caller",
			MessageKey: "msg",
			EncodeTime: zapcore.RFC3339TimeEncoder,
			LineEnding: zapcore.DefaultLineEnding,
			EncodeLevel: func() zapcore.LevelEncoder {
				if constants.IsDevelopment {
					return zapcore.CapitalColorLevelEncoder
				} else {
					return zapcore.CapitalLevelEncoder
				}
			}(),
			EncodeCaller:     zapcore.ShortCallerEncoder,
			ConsoleSeparator: " ",
		},
	}

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(logger)
}

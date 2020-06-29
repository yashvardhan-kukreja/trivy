package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/xerrors"
)

var (
	Logger      *zap.SugaredLogger
	debugOption bool
)

func InitLogger(debug, disable bool) (err error) {
	debugOption = debug
	Logger, err = NewLogger(debug, disable)
	if err != nil {
		return xerrors.Errorf("error in new logger: %w", err)
	}
	return nil

}

func NewLogger(debug, disable bool) (*zap.SugaredLogger, error) {
	// First, define our level-handling logic.
	errorPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	logPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		if debug {
			return lvl < zapcore.ErrorLevel
		}
		// Not enable debug level
		return zapcore.DebugLevel < lvl && lvl < zapcore.ErrorLevel
	})

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "Time",
		LevelKey:       "Level",
		NameKey:        "Name",
		CallerKey:      "Caller",
		MessageKey:     "Msg",
		StacktraceKey:  "St",
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

	// High-priority output should also go to standard error, and low-priority
	// output should also go to standard out.
	consoleLogs := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)
	if disable {
		devNull, err := os.Create(os.DevNull)
		if err != nil {
			return nil, err
		}
		// Discard low-priority output
		consoleLogs = zapcore.Lock(devNull)
	}

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleErrors, errorPriority),
		zapcore.NewCore(consoleEncoder, consoleLogs, logPriority),
	)

	opts := []zap.Option{zap.ErrorOutput(zapcore.Lock(os.Stderr))}
	if debug {
		opts = append(opts, zap.Development())
	}
	logger := zap.New(core, opts...)

	return logger.Sugar(), nil
}

func Fatal(err error) {
	if debugOption {
		Logger.Fatalf("%+v", err)
	}
	Logger.Fatal(err)
}

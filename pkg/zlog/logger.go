package zlog

import (
	"endpoint/pkg/config"
	"endpoint/pkg/model"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
	"time"
)

var (
	logger *zap.Logger
	levels = map[string]zapcore.Level{
		"debug": zap.DebugLevel,
		"info":  zap.InfoLevel,
		"warn":  zap.WarnLevel,
		"error": zap.ErrorLevel,
		"fatal": zap.FatalLevel,
	}
)

func Init(c *model.Log) error {
	developmentEncoderConfig := zap.NewDevelopmentEncoderConfig()
	developmentEncoderConfig.StacktraceKey = ""
	developmentEncoderConfig.EncodeCaller = nil
	developmentEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(developmentEncoderConfig)

	fileEncoderConfig := zap.NewProductionEncoderConfig()
	//fileEncoderConfig.StacktraceKey = ""
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	fileEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	fileEncoder := zapcore.NewConsoleEncoder(fileEncoderConfig)

	path := fmt.Sprintf("error_%s.log", time.Now().Format(time.DateOnly))
	if c != nil && c.LogFilePath != "" {
		path = filepath.Join([]string{c.LogFilePath, path}...)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
	if err != nil {
		return err
	}

	fL := config.DefaultFileLevel
	cL := config.DefaultConsoleLevel

	if c != nil && c.FileLevel != "" {
		fL = c.FileLevel
	}
	if c != nil && c.ConsoleLevel != "" {
		cL = c.ConsoleLevel
	}

	fileCore := zapcore.NewCore(
		fileEncoder,
		zapcore.AddSync(file),
		levels[fL],
	)

	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		levels[cL],
	)

	logger = zap.New(zapcore.NewTee(consoleCore, fileCore),
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	return nil
}

func Info(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	logger.Warn(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	logger.Debug(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
}

func Unwrap(err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
		logger.Error("", fields...)
	}
}

func UnwrapWithMessage(msg string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
		logger.Error("", fields...)
	}
}

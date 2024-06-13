package zlog

import (
	"endpoint/pkg/config"
	"endpoint/pkg/model"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	lock    sync.Mutex
	logger  *zap.Logger
	preFile *os.File
	levels  = map[string]zapcore.Level{
		"debug": zap.DebugLevel,
		"info":  zap.InfoLevel,
		"warn":  zap.WarnLevel,
		"error": zap.ErrorLevel,
		"fatal": zap.FatalLevel,
	}
	workDir, _ = os.Getwd()
)

func Init(c *model.Log) error {
	f := func() {
		developmentEncoderConfig := zap.NewDevelopmentEncoderConfig()
		developmentEncoderConfig.StacktraceKey = ""
		developmentEncoderConfig.EncodeCaller = nil
		developmentEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		consoleEncoder := zapcore.NewConsoleEncoder(developmentEncoderConfig)

		fileEncoderConfig := zap.NewProductionEncoderConfig()
		fileEncoderConfig.StacktraceKey = ""
		fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		fileEncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
		fileEncoderConfig.EncodeCaller = nil
		fileEncoder := zapcore.NewConsoleEncoder(fileEncoderConfig)

		file, err := os.OpenFile(fmt.Sprintf("%s/error_%s.log", workDir, time.Now().Format(time.DateOnly)), os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
		if err != nil {
			log.Println(err)
			return
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

		lock.Lock()
		defer lock.Unlock()

		logger = zap.New(zapcore.NewTee(consoleCore, fileCore),
			zap.AddCaller(),
			zap.AddCallerSkip(1),
			zap.AddStacktrace(zapcore.ErrorLevel),
		)

		if preFile != nil {
			_ = preFile.Close()
		}

		preFile = file
	}

	f()

	go func() {
		for {
			now := time.Now()
			nextDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			time.Sleep(time.Until(nextDay))

			f()
		}
	}()
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

func Ignore(err error) bool {
	if strings.Contains(err.Error(), closeStreamErr) || strings.Contains(err.Error(), opErr) ||
		strings.Contains(err.Error(), noPeerResp) || strings.Contains(err.Error(), finalSizeErr) {
		return true
	}
	return false
}

var (
	closeStreamErr = "read from closed stream"
	opErr          = "use of closed network connection"
	noPeerResp     = "peer did not respond to CONNECTION_CLOSE"
	finalSizeErr   = "end of stream occurs before prior data"
)

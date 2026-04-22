package log

import (
	"fmt"
	"github.com/GMWalletApp/epusdt/config"
	"github.com/natefinch/lumberjack"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

var Sugar *zap.SugaredLogger

func Init() {
	level := getLogLevel()
	core := zapcore.NewTee(
		zapcore.NewCore(getConsoleEncoder(), getConsoleWriter(), level),
		zapcore.NewCore(getFileEncoder(), getFileWriter(), level),
	)
	logger := zap.New(core, zap.AddCaller())
	Sugar = logger.Sugar()
}

func getFileEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getConsoleEncoder() zapcore.Encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getFileWriter() zapcore.WriteSyncer {
	file := fmt.Sprintf("%s/log_%s.log",
		config.LogSavePath,
		time.Now().Format("20060102"))
	lumberJackLogger := &lumberjack.Logger{
		Filename:   file,
		MaxSize:    viper.GetInt("log_max_size"),
		MaxBackups: viper.GetInt("max_backups"),
		MaxAge:     viper.GetInt("log_max_age"),
		Compress:   false,
	}
	return zapcore.AddSync(lumberJackLogger)
}

func getConsoleWriter() zapcore.WriteSyncer {
	return zapcore.Lock(os.Stdout)
}

func getLogLevel() zapcore.Level {
	switch config.LogLevel {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

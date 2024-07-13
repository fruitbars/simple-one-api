package mylog

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
)

var Logger *zap.Logger

func InitLog(mode string) {

	log.Println("level mode", mode)
	var encoder zapcore.Encoder
	var encoderConfig zapcore.EncoderConfig
	var level zapcore.Level

	// 根据模式选择合适的编码器配置和日志级别
	switch mode {
	case "prod", "production", "prodj", "prodjson", "productionjson":
		encoderConfig = zap.NewProductionEncoderConfig()
		level = zapcore.WarnLevel
	case "dev", "development":
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		level = zapcore.InfoLevel
	case "debug":
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		level = zapcore.DebugLevel
	default:
		log.Println("level mode default prod")
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		level = zapcore.WarnLevel
	}

	// 设置时间键和时间格式
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// 根据需要选择输出为JSON或控制台格式
	if mode == "prodj" || mode == "prodjson" || mode == "productionjson" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
		log.Println("log json format")
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
		log.Println("log plain-text format")
	}

	// 创建日志核心
	core := zapcore.NewCore(
		encoder,
		zapcore.Lock(os.Stdout),
		zap.NewAtomicLevelAt(level),
	)

	// 构建日志器
	Logger = zap.New(core, zap.AddCaller())
}

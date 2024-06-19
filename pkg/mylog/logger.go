// log/logger.go
package log

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"runtime"
	"time"
)

// 创建一个全局的 logrus 实例
var Logger = logrus.New()

func InitLog(logLevel logrus.Level) {
	// 配置日志等级
	Logger.SetLevel(logLevel)
	// 配置日志格式
	Logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,         // 输出完整的时间戳
		TimestampFormat:  time.RFC3339, // 时间戳格式
		DisableColors:    true,         // 禁用颜色输出
		ForceColors:      true,         // 强制启用颜色输出，即使输出不支持
		DisableTimestamp: false,        // 禁用时间戳
		QuoteEmptyFields: true,         // 将空字段用引号括起来
		CallerPrettyfier: func(f *runtime.Frame) (string, string) { // 自定义调用者信息
			return fmt.Sprintf("func: %s", f.Function), fmt.Sprintf("file: %s", f.File)
		},
	})
	// 配置输出到标准输出
	Logger.SetOutput(os.Stdout)
}

// 封装日志方法
func Info(args ...interface{}) {
	Logger.Info(args...)
}

func Warn(args ...interface{}) {
	Logger.Warn(args...)
}

func Error(args ...interface{}) {
	Logger.Error(args...)
}

func Debug(args ...interface{}) {
	Logger.Debug(args...)
}

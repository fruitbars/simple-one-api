package mycommon

import (
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mycomdef"
)

// 通用的限流器详情获取函数
func getLimitDetails(limit config.Limit) (string, float64, int) {
	switch {
	case limit.QPS > 0:
		return mycomdef.KEYNAME_QPS, limit.QPS, limit.Timeout
	case limit.QPM > 0:
		return mycomdef.KEYNAME_QPM, limit.QPM, limit.Timeout
	case limit.RPM > 0:
		return mycomdef.KEYNAME_QPM, limit.RPM, limit.Timeout
	case limit.Concurrency > 0:
		return mycomdef.KEYNAME_CONCURRENCY, limit.Concurrency, limit.Timeout
	default:
		return "", 0, 0 // 默认返回
	}
}

// 获取服务模型的限流详情
func GetServiceModelDetailsLimit(s *config.ModelDetails) (string, float64, int) {
	return getLimitDetails(s.Limit)
}

// 获取服务限流器的限流详情
func GetServiceLimiterDetailsLimit(l *config.Limit) (string, float64, int) {
	return getLimitDetails(*l)
}

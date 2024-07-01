package mycommon

import (
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mycomdef"
)

func GetServiceModelDetailsLimit(s *config.ModelDetails) (limitType string, limitn float64, timeout int) {
	// 假设'limit'键下是一个JSON表示的map
	if s.Limit.QPS > 0 {
		return mycomdef.KEYNAME_QPS, s.Limit.QPS, s.Limit.Timeout
	} else if s.Limit.QPM > 0 {
		return mycomdef.KEYNAME_QPS, s.Limit.QPM / 60, s.Limit.Timeout
	} else if s.Limit.RPM > 0 {
		return mycomdef.KEYNAME_QPS, s.Limit.QPM / 60, s.Limit.Timeout
	} else if s.Limit.Concurrency > 0 {
		return mycomdef.KEYNAME_CONCURRENCY, s.Limit.Concurrency, s.Limit.Timeout
	}

	return "", 0, 0 // 默认返回
}

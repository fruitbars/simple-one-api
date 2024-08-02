package mycommon

import (
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mycomdef"
	"strconv"
)

// GetACredentials 根据模型名从ModelDetails中选择合适的凭证
func GetACredentials(s *config.ModelDetails, model string) (map[string]interface{}, string) {
	// 检查是否有多个凭据列表可用
	var credID string
	if s.CredentialList != nil && len(s.CredentialList) > 0 {
		key := s.ServiceID + "credentials"

		index := config.GetLBIndex(config.LoadBalancingStrategy, key, len(s.CredentialList))
		credID = s.ServiceID + "_credentials_" + strconv.Itoa(index)
		return s.CredentialList[index], credID
	}
	return s.Credentials, credID
}

func GetCredentialLimit(credentials map[string]interface{}) (limitType string, limitn float64, timeout int) {
	// 假设'limit'键下是一个JSON表示的map
	limitData, ok := credentials["limit"].(map[string]interface{})
	if !ok {
		return "", 0, 0 // 没有找到或类型不匹配
	}

	if to, ok := limitData["timeout"].(int); ok {
		timeout = to
	}
	// 按优先级查找限制值：qps, qpm, rpm, concurrency
	if qps, ok := limitData[mycomdef.KEYNAME_QPS].(float64); ok {
		return mycomdef.KEYNAME_QPS, qps, timeout
	}
	if qpm, ok := limitData[mycomdef.KEYNAME_QPM].(float64); ok {
		return mycomdef.KEYNAME_QPM, qpm, timeout
	}
	if rpm, ok := limitData[mycomdef.KEYNAME_RPM].(float64); ok {
		return mycomdef.KEYNAME_QPM, rpm, timeout
	}
	if concurrency, ok := limitData[mycomdef.KEYNAME_CONCURRENCY].(float64); ok {
		return mycomdef.KEYNAME_CONCURRENCY, concurrency, timeout
	}

	return "", 0, 0 // 默认返回
}

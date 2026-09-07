package mycommon

import (
	"encoding/json"
	"fmt"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mycomdef"
	"strings"
)

type CredentialSelection struct {
	Credentials map[string]interface{}
	ID          string
	Name        string
}

// GetCredentialCandidates returns an ordered, healthy view of a provider's
// credential pool. The first item follows the global load-balancing strategy;
// subsequent items provide deterministic failover within the same pool.
func GetCredentialCandidates(s *config.ModelDetails, model string) []CredentialSelection {
	if s == nil {
		return nil
	}
	if len(s.CredentialList) == 0 {
		return []CredentialSelection{{Credentials: s.Credentials, ID: s.ServiceID}}
	}
	start := config.GetLBIndex(config.CurrentLoadBalancing(), s.ServiceID+":credentials", len(s.CredentialList))
	candidates := make([]CredentialSelection, 0, len(s.CredentialList))
	for offset := 0; offset < len(s.CredentialList); offset++ {
		index := (start + offset) % len(s.CredentialList)
		credential := s.CredentialList[index]
		if enabled, ok := credential["enabled"].(bool); ok && !enabled {
			continue
		}
		credentialID, _ := credential["id"].(string)
		credentialID = strings.TrimSpace(credentialID)
		if credentialID == "" {
			credentialID = fmt.Sprintf("key-%d", index+1)
		}
		name, _ := credential["name"].(string)
		id := s.ServiceID + ":" + credentialID
		if config.CircuitBreakerAvailable(id, model) {
			candidates = append(candidates, CredentialSelection{Credentials: credential, ID: id, Name: strings.TrimSpace(name)})
		}
	}
	return candidates
}

// GetACredentials 根据模型名从ModelDetails中选择合适的凭证
func GetACredentials(s *config.ModelDetails, model string) (map[string]interface{}, string) {
	candidates := GetCredentialCandidates(s, model)
	if len(candidates) > 0 {
		return candidates[0].Credentials, candidates[0].ID
	}
	if s == nil {
		return nil, ""
	}
	return s.Credentials, s.ServiceID
}

func GetCredentialLimit(credentials map[string]interface{}) (limitType string, limitn float64, timeout int) {
	limit := GetCredentialLimits(credentials)
	timeout = limit.Timeout
	// 按优先级查找限制值：qps, qpm, rpm, concurrency
	if limit.QPS > 0 {
		return mycomdef.KEYNAME_QPS, limit.QPS, timeout
	}
	if limit.QPM > 0 {
		return mycomdef.KEYNAME_QPM, limit.QPM, timeout
	}
	if limit.RPM > 0 {
		return mycomdef.KEYNAME_RPM, limit.RPM, timeout
	}
	if limit.Concurrency > 0 {
		return mycomdef.KEYNAME_CONCURRENCY, limit.Concurrency, timeout
	}

	return "", 0, 0 // 默认返回
}

func GetCredentialLimits(credentials map[string]interface{}) config.Limit {
	if credentials == nil {
		return config.Limit{}
	}
	raw, exists := credentials["limit"]
	if !exists {
		return config.Limit{}
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return config.Limit{}
	}
	var limit config.Limit
	if json.Unmarshal(payload, &limit) != nil {
		return config.Limit{}
	}
	return limit
}

// GetCredentialModelLimit returns the limit for one model on a credential.
// Model limits are scoped by the credential ID at call sites.
func GetCredentialModelLimit(credentials map[string]interface{}, modelNames ...string) (config.Limit, string, bool) {
	if credentials == nil {
		return config.Limit{}, "", false
	}
	limits := make(map[string]config.Limit)
	raw, exists := credentials["model_limits"]
	if !exists || raw == nil {
		return config.Limit{}, "", false
	}
	payload, err := json.Marshal(raw)
	if err != nil || json.Unmarshal(payload, &limits) != nil {
		return config.Limit{}, "", false
	}
	for _, name := range modelNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if limit, ok := limits[name]; ok {
			return limit, name, true
		}
	}
	return config.Limit{}, "", false
}

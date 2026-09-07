package apis

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mycommon"
)

// CredentialCapacityStatus is the runtime, secret-free capacity view shown by
// the admin UI. One row is returned for each configured credential/model pair.
type CredentialCapacityStatus struct {
	ProviderID      string     `json:"provider_id"`
	ProviderName    string     `json:"provider_name"`
	CredentialID    string     `json:"credential_id"`
	CredentialName  string     `json:"credential_name"`
	Model           string     `json:"model"`
	TPMLimit        float64    `json:"tpm_limit"`
	ReservedTokens  int64      `json:"reserved_tokens"`
	RemainingTokens int64      `json:"remaining_tokens"`
	Available       bool       `json:"available"`
	CooldownUntil   *time.Time `json:"cooldown_until,omitempty"`
	AvailableAt     *time.Time `json:"available_at,omitempty"`
}

func AdminCapacityHandler(c *gin.Context) {
	conf := config.CurrentConfiguration()
	serviceNames := make([]string, 0, len(conf.Services))
	for serviceName := range conf.Services {
		serviceNames = append(serviceNames, serviceName)
	}
	sort.Strings(serviceNames)

	result := make([]CredentialCapacityStatus, 0)
	for _, serviceName := range serviceNames {
		services := conf.Services[serviceName]
		for _, service := range services {
			if !service.Enabled || strings.TrimSpace(service.ID) == "" {
				continue
			}
			models := append([]string(nil), service.Models...)
			if len(models) == 0 {
				continue
			}
			if len(service.CredentialList) == 0 {
				if len(service.Credentials) == 0 {
					continue
				}
				for _, model := range models {
					result = append(result, capacityStatusForCredential(serviceName, service.ID, service.Provider, service.ID, "主 Key", service.Credentials, model))
				}
				continue
			}
			for index, credential := range service.CredentialList {
				if enabled, ok := credential["enabled"].(bool); ok && !enabled {
					continue
				}
				credentialID, _ := credential["id"].(string)
				if strings.TrimSpace(credentialID) == "" {
					credentialID = "key-" + strconv.Itoa(index+1)
				}
				credentialName, _ := credential["name"].(string)
				if strings.TrimSpace(credentialName) == "" {
					credentialName = "Key " + strconv.Itoa(index+1)
				}
				for _, model := range models {
					result = append(result, capacityStatusForCredential(serviceName, service.ID, service.Provider, credentialID, credentialName, credential, model))
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func capacityStatusForCredential(serviceName, providerID, providerName, credentialID, credentialName string, credentials map[string]interface{}, model string) CredentialCapacityStatus {
	schedulerID := providerID
	if credentialID != providerID {
		schedulerID += ":" + credentialID
	}
	snapshot := mycommon.InspectCredentialCapacity(credentials, schedulerID, model)
	return CredentialCapacityStatus{
		ProviderID: providerID, ProviderName: firstNonEmpty(strings.TrimSpace(providerName), serviceName),
		CredentialID: credentialID, CredentialName: credentialName, Model: model,
		TPMLimit: snapshot.TPMLimit, ReservedTokens: snapshot.ReservedTokens,
		RemainingTokens: snapshot.RemainingTokens, Available: snapshot.Available,
		CooldownUntil: snapshot.CooldownUntil, AvailableAt: snapshot.AvailableAt,
	}
}

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

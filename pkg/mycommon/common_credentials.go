package mycommon

import (
	"math/rand"
	"simple-one-api/pkg/config"
)

func GetACredentials(s *config.ModelDetails) map[string]string {
	var credentials map[string]string
	if s.CredentialList != nil && len(s.CredentialList) > 0 {
		credentials = s.CredentialList[rand.Intn(len(s.CredentialList))]
	} else {
		credentials = s.Credentials
	}

	return credentials
}

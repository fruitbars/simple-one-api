package mycommon

import (
	"encoding/json"
	"testing"
)

func TestGetCredentialLimitAcceptsJSONFloatTimeout(t *testing.T) {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(`{"limit":{"qps":2,"timeout":12}}`), &credentials); err != nil {
		t.Fatalf("decode credentials: %v", err)
	}

	limitType, limit, timeout := GetCredentialLimit(credentials)
	if limitType != "qps" {
		t.Fatalf("unexpected limit type: %q", limitType)
	}
	if limit != 2 {
		t.Fatalf("unexpected limit value: %v", limit)
	}
	if timeout != 12 {
		t.Fatalf("JSON-decoded timeout must be preserved, got %d", timeout)
	}
}

package initializer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"simple-one-api/pkg/config"
	"simple-one-api/pkg/configstore"
)

func TestSetupImportsFileConfigurationIntoSQLite(t *testing.T) {
	resetInitializerForTest(t)
	dbPath := filepath.Join(t.TempDir(), "config.db")
	t.Setenv("SIMPLE_ONE_API_DB", dbPath)
	configPath := writeConfigForTest(t, config.Configuration{
		APIKey:        "file-key",
		ServerPort:    ":19091",
		LoadBalancing: "first",
		Services:      testServices("model-a"),
	})

	if err := Setup(configPath); err != nil {
		t.Fatalf("setup: %v", err)
	}

	store := ConfigStore()
	if store == nil {
		t.Fatal("expected SQLite store")
	}
	revisions, err := store.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(revisions) != 1 || !revisions[0].Active || revisions[0].Source != "legacy-file" {
		t.Fatalf("unexpected imported revisions: %#v", revisions)
	}
	if config.CurrentAPIKey() != "file-key" {
		t.Fatalf("expected file config to be active, got %q", config.CurrentAPIKey())
	}
	if info, err := os.Stat(dbPath); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("database permissions = %v, %v; want 0600", info, err)
	}
}

func TestSetupStartsWithBuiltInDefaultsWhenDefaultConfigIsMissing(t *testing.T) {
	resetInitializerForTest(t)
	directory := t.TempDir()
	dbPath := filepath.Join(directory, "config.db")
	t.Setenv("SIMPLE_ONE_API_DB", dbPath)
	previousWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previousWorkingDirectory) })

	if err := Setup("config.json"); err != nil {
		t.Fatalf("setup without default config: %v", err)
	}

	if config.CurrentServerPort() != ":9090" {
		t.Fatalf("expected built-in default port, got %q", config.CurrentServerPort())
	}
	revisions, err := ConfigStore().List(context.Background(), 10)
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(revisions) != 1 || !revisions[0].Active || revisions[0].Source != "bootstrap-default" {
		t.Fatalf("unexpected bootstrap revisions: %#v", revisions)
	}
}

func TestSetupRestoresSQLiteActiveRevisionWhenDefaultConfigIsMissing(t *testing.T) {
	resetInitializerForTest(t)
	directory := t.TempDir()
	dbPath := filepath.Join(directory, "config.db")
	t.Setenv("SIMPLE_ONE_API_DB", dbPath)
	previousWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previousWorkingDirectory) })

	store, err := configstore.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	activePayload, err := json.Marshal(config.Configuration{
		APIKey:        "db-key",
		ServerPort:    ":19089",
		LoadBalancing: "first",
		Services:      testServices("model-from-db"),
	})
	if err != nil {
		t.Fatalf("marshal active revision: %v", err)
	}
	if _, err := store.CreateRevision(context.Background(), activePayload, "admin", "db active", true); err != nil {
		t.Fatalf("create active revision: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	if err := Setup("config.json"); err != nil {
		t.Fatalf("setup without default config: %v", err)
	}

	if config.CurrentAPIKey() != "db-key" {
		t.Fatalf("expected SQLite active revision, got API key %q", config.CurrentAPIKey())
	}
	if _, ok := config.CurrentSupportModels()["model-from-db"]; !ok {
		t.Fatalf("expected runtime snapshot from SQLite active revision, got %#v", config.CurrentSupportModels())
	}
}

func TestSetupKeepsInitializationErrorAfterOnce(t *testing.T) {
	resetInitializerForTest(t)
	configPath := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(configPath, []byte(`{"services":`), 0o600); err != nil {
		t.Fatalf("write broken config: %v", err)
	}
	firstErr := Setup(configPath)
	secondErr := Setup(configPath)
	if firstErr == nil || secondErr == nil {
		t.Fatalf("setup error must remain visible after sync.Once: first=%v second=%v", firstErr, secondErr)
	}
}

func TestSetupFailsWhenSQLiteRepositoryCannotOpen(t *testing.T) {
	resetInitializerForTest(t)
	configPath := writeConfigForTest(t, config.Configuration{ServerPort: ":19090"})
	blockedParent := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blockedParent, []byte("blocked"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	t.Setenv("SIMPLE_ONE_API_DB", filepath.Join(blockedParent, "config.db"))

	if err := Setup(configPath); err == nil {
		t.Fatal("setup must fail when the configured SQLite repository cannot open")
	}
	if ConfigStore() != nil {
		t.Fatal("failed setup must not expose a partial SQLite store")
	}
}

func TestSetupRestoresSQLiteActiveRevisionWhenFileUnchanged(t *testing.T) {
	resetInitializerForTest(t)
	dbPath := filepath.Join(t.TempDir(), "config.db")
	t.Setenv("SIMPLE_ONE_API_DB", dbPath)
	configPath := writeConfigForTest(t, config.Configuration{
		APIKey:        "file-key",
		ServerPort:    ":19092",
		LoadBalancing: "first",
		Services:      testServices("model-a"),
	})
	if err := Setup(configPath); err != nil {
		t.Fatalf("initial setup: %v", err)
	}
	activeDBPayload, err := json.Marshal(config.Configuration{
		APIKey:        "db-key",
		ServerPort:    ":19092",
		LoadBalancing: "first",
		Services:      testServices("model-b"),
	})
	if err != nil {
		t.Fatalf("marshal active revision: %v", err)
	}
	if _, err := ConfigStore().CreateRevision(context.Background(), activeDBPayload, "admin", "db active", true); err != nil {
		t.Fatalf("create active db revision: %v", err)
	}

	resetInitializerForTest(t)
	t.Setenv("SIMPLE_ONE_API_DB", dbPath)
	if err := Setup(configPath); err != nil {
		t.Fatalf("restart setup: %v", err)
	}

	if config.CurrentAPIKey() != "db-key" {
		t.Fatalf("expected unchanged file to restore SQLite active revision, got %q", config.CurrentAPIKey())
	}
	if _, ok := config.CurrentSupportModels()["model-b"]; !ok {
		t.Fatalf("expected runtime snapshot from SQLite active revision, got %#v", config.CurrentSupportModels())
	}
}

func TestSetupImportsNewRevisionWhenFileChecksumChanges(t *testing.T) {
	resetInitializerForTest(t)
	dbPath := filepath.Join(t.TempDir(), "config.db")
	t.Setenv("SIMPLE_ONE_API_DB", dbPath)
	configPath := writeConfigForTest(t, config.Configuration{
		APIKey:        "old-file-key",
		ServerPort:    ":19093",
		LoadBalancing: "first",
		Services:      testServices("model-a"),
	})
	if err := Setup(configPath); err != nil {
		t.Fatalf("initial setup: %v", err)
	}
	overwriteConfigForTest(t, configPath, config.Configuration{
		APIKey:        "new-file-key",
		ServerPort:    ":19093",
		LoadBalancing: "first",
		Services:      testServices("model-c"),
	})

	resetInitializerForTest(t)
	t.Setenv("SIMPLE_ONE_API_DB", dbPath)
	if err := Setup(configPath); err != nil {
		t.Fatalf("restart setup: %v", err)
	}

	revisions, err := ConfigStore().List(context.Background(), 10)
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(revisions) != 2 || !revisions[0].Active {
		t.Fatalf("expected changed file to create a second active revision, got %#v", revisions)
	}
	if config.CurrentAPIKey() != "new-file-key" {
		t.Fatalf("expected changed file config to be active, got %q", config.CurrentAPIKey())
	}
}

func TestPublishAndActivateConfigurationUpdateRuntimeSnapshot(t *testing.T) {
	resetInitializerForTest(t)
	store, err := configstore.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	repository = store
	t.Cleanup(func() {
		store.Close()
		repository = nil
	})
	first := config.Configuration{
		APIKey:        "first-key",
		ServerPort:    ":19094",
		LoadBalancing: "first",
		Services:      testServices("model-a"),
	}
	if err := config.ApplyConfiguration(first, "test.json"); err != nil {
		t.Fatalf("apply first config: %v", err)
	}
	firstPayload, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first config: %v", err)
	}
	firstRevision, err := store.CreateRevision(context.Background(), firstPayload, "test", "first", true)
	if err != nil {
		t.Fatalf("create first revision: %v", err)
	}

	second := first
	second.APIKey = "second-key"
	second.Services = testServices("model-b")
	published, err := PublishConfiguration(context.Background(), second, "admin", "second")
	if err != nil {
		t.Fatalf("publish second config: %v", err)
	}
	if !published.Active || config.CurrentAPIKey() != "second-key" {
		t.Fatalf("publish did not activate runtime snapshot: %#v, key=%q", published, config.CurrentAPIKey())
	}
	if _, ok := config.CurrentSupportModels()["model-b"]; !ok {
		t.Fatalf("expected published model in runtime snapshot: %#v", config.CurrentSupportModels())
	}

	activated, err := ActivateConfiguration(context.Background(), firstRevision.ID)
	if err != nil {
		t.Fatalf("activate first config: %v", err)
	}
	if !activated.Active || config.CurrentAPIKey() != "first-key" {
		t.Fatalf("activate did not restore runtime snapshot: %#v, key=%q", activated, config.CurrentAPIKey())
	}
	if _, ok := config.CurrentSupportModels()["model-a"]; !ok {
		t.Fatalf("expected activated model in runtime snapshot: %#v", config.CurrentSupportModels())
	}
}

func resetInitializerForTest(t *testing.T) {
	t.Helper()
	if repository != nil {
		_ = repository.Close()
	}
	once = sync.Once{}
	setupErr = nil
	repository = nil
}

func writeConfigForTest(t *testing.T, conf config.Configuration) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	overwriteConfigForTest(t, path, conf)
	return path
}

func overwriteConfigForTest(t *testing.T, path string, conf config.Configuration) {
	t.Helper()
	payload, err := json.Marshal(conf)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func testServices(model string) map[string][]config.ServiceModel {
	return map[string][]config.ServiceModel{
		"openai": {{
			Provider:    "openai",
			Enabled:     true,
			Models:      []string{model},
			Credentials: map[string]interface{}{"api_key": "provider-key"},
		}},
	}
}

// pkg/initializer/initializer.go
package initializer

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"path/filepath"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/configstore"
	"simple-one-api/pkg/mylog"
	"strings"
	"sync"
)

var once sync.Once
var setupErr error
var repository *configstore.Store
var publishMu sync.Mutex

const legacyChecksumKey = "legacy_config_checksum"

// Setup initializes the configuration and logging system.
func Setup(configName string) error {
	once.Do(func() {
		setupErr = config.InitConfig(configName)
		if setupErr != nil {
			log.Println("Error initializing config:", setupErr)
			return
		}

		log.Println("config.InitConfig ok")
		if repository, setupErr = openConfigurationRepository(config.CurrentConfigPath()); setupErr != nil {
			repository = nil
		} else if setupErr = reconcileConfiguration(context.Background(), repository); setupErr != nil {
			_ = repository.Close()
			repository = nil
			return
		}
		if setupErr != nil {
			return
		}

		if !config.CurrentDebug() {
			gin.SetMode(gin.ReleaseMode)
		}

		mylog.InitLog(config.CurrentLogLevel())
		log.Println("config.LogLevel ok")
	})
	return setupErr
}

func openConfigurationRepository(configPath string) (*configstore.Store, error) {
	databasePath := strings.TrimSpace(os.Getenv("SIMPLE_ONE_API_DB"))
	if databasePath == "" {
		extension := filepath.Ext(configPath)
		databasePath = strings.TrimSuffix(configPath, extension) + ".db"
	}
	return configstore.Open(databasePath)
}

func reconcileConfiguration(ctx context.Context, store *configstore.Store) error {
	filePayload, err := os.ReadFile(config.CurrentConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return reconcileMissingFileConfiguration(ctx, store)
		}
		return err
	}
	fileChecksum := configstore.Checksum(filePayload)
	lastChecksum, hasChecksum, err := store.Metadata(ctx, legacyChecksumKey)
	if err != nil {
		return err
	}
	_, activePayload, activeErr := store.Active(ctx)
	if activeErr == nil && hasChecksum && lastChecksum == fileChecksum {
		var active config.Configuration
		if err := json.Unmarshal(activePayload, &active); err != nil {
			return err
		}
		return config.ApplyConfiguration(active, config.CurrentConfigPath())
	}
	if activeErr != nil && activeErr != configstore.ErrNoActiveRevision {
		return activeErr
	}
	normalized, err := json.Marshal(config.CurrentConfiguration())
	if err != nil {
		return err
	}
	if _, err := store.CreateRevision(ctx, normalized, "legacy-file", "Imported from "+config.CurrentConfigPath(), true); err != nil {
		return err
	}
	return store.SetMetadata(ctx, legacyChecksumKey, fileChecksum)
}

func reconcileMissingFileConfiguration(ctx context.Context, store *configstore.Store) error {
	_, activePayload, activeErr := store.Active(ctx)
	if activeErr == nil {
		var active config.Configuration
		if err := json.Unmarshal(activePayload, &active); err != nil {
			return err
		}
		return config.ApplyConfiguration(active, config.CurrentConfigPath())
	}
	if activeErr != nil && activeErr != configstore.ErrNoActiveRevision {
		return activeErr
	}
	normalized, err := json.Marshal(config.CurrentConfiguration())
	if err != nil {
		return err
	}
	_, err = store.CreateRevision(ctx, normalized, "bootstrap-default", "Created from built-in defaults because config file was not found", true)
	return err
}

func ConfigStore() *configstore.Store { return repository }

func PublishConfiguration(ctx context.Context, conf config.Configuration, source, note string) (configstore.Revision, error) {
	publishMu.Lock()
	defer publishMu.Unlock()
	prepared, err := config.PrepareConfiguration(conf, config.CurrentConfigPath())
	if err != nil {
		if issues := config.ValidateConfiguration(conf); len(issues) > 0 {
			return configstore.Revision{}, &ConfigurationValidationError{Issues: issues}
		}
		return configstore.Revision{}, err
	}
	// Persist the exact normalized configuration that will become the runtime
	// snapshot. This makes generated service IDs and normalized values durable
	// across restarts instead of recalculating them from the raw draft.
	payload, err := json.Marshal(prepared.Configuration())
	if err != nil {
		return configstore.Revision{}, err
	}
	if repository == nil {
		return configstore.Revision{}, configstore.ErrNoActiveRevision
	}
	revision, err := repository.CreateRevision(ctx, payload, source, note, true)
	if err != nil {
		return configstore.Revision{}, err
	}
	prepared.Publish()
	return revision, nil
}

func ActivateConfiguration(ctx context.Context, id int64) (configstore.Revision, error) {
	publishMu.Lock()
	defer publishMu.Unlock()
	if repository == nil {
		return configstore.Revision{}, configstore.ErrNoActiveRevision
	}
	_, payload, err := repository.Revision(ctx, id)
	if err != nil {
		return configstore.Revision{}, err
	}
	var conf config.Configuration
	if err := json.Unmarshal(payload, &conf); err != nil {
		return configstore.Revision{}, err
	}
	prepared, err := config.PrepareConfiguration(conf, config.CurrentConfigPath())
	if err != nil {
		if issues := config.ValidateConfiguration(conf); len(issues) > 0 {
			return configstore.Revision{}, &ConfigurationValidationError{Issues: issues}
		}
		return configstore.Revision{}, err
	}
	revision, _, err := repository.Activate(ctx, id)
	if err != nil {
		return configstore.Revision{}, err
	}
	prepared.Publish()
	return revision, nil
}

type ConfigurationValidationError struct {
	Issues []config.ValidationIssue
}

func (e *ConfigurationValidationError) Error() string {
	if len(e.Issues) == 0 {
		return "configuration validation failed"
	}
	return e.Issues[0].Path + ": " + e.Issues[0].Message
}

func Cleanup() {
	if repository != nil {
		repository.Close()
	}
	if mylog.Logger != nil {
		mylog.Logger.Sync() // Ensure all logs are flushed properly
	}
}

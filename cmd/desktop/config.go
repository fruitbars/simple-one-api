package main

import (
	"errors"
	"os"
	"path/filepath"
)

const defaultDesktopConfig = `{
  "server_port": ":9090",
  "log_level": "info",
  "enable_web": true,
  "services": {}
}
`

func resolveDesktopConfig(args []string, userConfigDir string) (string, error) {
	if len(args) > 1 {
		return filepath.Abs(args[1])
	}
	if userConfigDir == "" {
		return "", errors.New("user configuration directory is unavailable")
	}

	directory := filepath.Join(userConfigDir, "simple-one-api")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	configPath := filepath.Join(directory, "config.json")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.WriteFile(configPath, []byte(defaultDesktopConfig), 0o600); err != nil {
		return "", err
	}
	return configPath, nil
}

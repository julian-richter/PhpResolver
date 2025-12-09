package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	configDirName  = ".phpResolver"
	configFileName = "config.yml"
)

func defaultConfig() Config {
	return Config{
		Log: LogConfig{
			Level:       LogLevelInfo,
			Format:      LogFormatText,
			ShowSource:  true,
			FileEnabled: false,
			FilePath:    "",
		},
	}
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, configDirName, configFileName), nil
}

func ensureConfigFile(path string) (Config, error) {
	// IMPORTANT: Uses defaultConfig() as base, then yaml.Unmarshal merges
	// file values into it. This preserves defaults for any absent YAML fields.
	// Logic intentionally mixes config creation + loading for simplicity;
	cfg := defaultConfig()

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return cfg, fmt.Errorf("create config dir: %w", err)
		}
		data, err := yaml.Marshal(&cfg)
		if err != nil {
			return cfg, fmt.Errorf("marshal default config: %w", err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return cfg, fmt.Errorf("write default config: %w", err)
		}
		return cfg, nil
	} else if err != nil {
		return cfg, fmt.Errorf("stat config: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	// yaml.Unmarshal merges into pre-populated cfg, preserving defaults for missing keys
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

func Load() (Config, error) {
	p, err := configPath()
	if err != nil {
		return Config{}, err
	}
	cfg, err := ensureConfigFile(p)
	if err != nil {
		return Config{}, err
	}
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func validate(cfg Config) error {
	if !IsValidLogLevel(cfg.Log.Level) {
		return fmt.Errorf("invalid log.level %q (must be one of: %v)",
			cfg.Log.Level, ValidLogLevels())
	}

	if !IsValidLogFormat(cfg.Log.Format) {
		return fmt.Errorf("invalid log.format %q (must be one of: %v)",
			cfg.Log.Format, ValidLogFormats())
	}

	return nil
}

// internal/config/logger.go
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	charmLog "github.com/charmbracelet/log"
)

func NewLogger(cfg Config) (*charmLog.Logger, error) {
	var level charmLog.Level
	switch cfg.Log.Level {
	case "debug":
		level = charmLog.DebugLevel
	case "info":
		level = charmLog.InfoLevel
	case "warn":
		level = charmLog.WarnLevel
	case "error":
		level = charmLog.ErrorLevel
	default:
		return nil, fmt.Errorf("unsupported log level %q", cfg.Log.Level)
	}

	var formatter charmLog.Formatter
	switch cfg.Log.Format {
	case LogFormatText:
		formatter = charmLog.TextFormatter
	case LogFormatJSON:
		formatter = charmLog.JSONFormatter
	case LogFormatLogfmt:
		formatter = charmLog.LogfmtFormatter
	default:
		return nil, fmt.Errorf("unsupported log format %q", cfg.Log.Format)
	}

	var writer io.Writer = os.Stderr

	if cfg.Log.FileEnabled {
		path := cfg.Log.FilePath
		if path == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("resolve home dir for log file: %w", err)
			}
			path = filepath.Join(home, ".phpResolver", "logs", "app.log")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("create log dir: %w", err)
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("open log file: %w", err)
		}
		writer = io.MultiWriter(os.Stderr, f)
	}

	logger := charmLog.NewWithOptions(writer, charmLog.Options{
		Level:           level,
		ReportTimestamp: true,
		ReportCaller:    cfg.Log.ShowSource,
		Formatter:       formatter,
		Prefix:          "phpResolver",
	})

	return logger, nil
}

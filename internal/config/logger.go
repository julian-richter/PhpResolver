// internal/config/logger.go
package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

const logPrefix = "phpResolver"

// NewLogger creates a configured charmbracelet/log.Logger from the config.
// Assumes config has already been validated by config.Load().
// Returns LoggerHandle with Closer to prevent file descriptor leaks.
func NewLogger(cfg Config) (*LoggerHandle, error) {

	// Map typed LogLevel to charm log.Level using constants
	var level log.Level
	switch cfg.Log.Level {
	case LogLevelDebug:
		level = log.DebugLevel
	case LogLevelInfo:
		level = log.InfoLevel
	case LogLevelWarn:
		level = log.WarnLevel
	case LogLevelError:
		level = log.ErrorLevel
	default:
		// Unreachable due to validation in config.Load()
		panic("unreachable: invalid log level")
	}

	// Map typed LogFormat to charm formatter using constants
	var formatter log.Formatter
	switch cfg.Log.Format {
	case LogFormatText:
		formatter = log.TextFormatter
	case LogFormatJSON:
		formatter = log.JSONFormatter
	case LogFormatLogfmt:
		formatter = log.LogfmtFormatter
	default:
		// Unreachable due to validation in config.Load()
		panic("unreachable: invalid log format")
	}

	// Default to stderr (no close needed)
	var file *os.File
	writer := io.Writer(os.Stderr)

	// Optional file output with auto-default path
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
			return nil, fmt.Errorf("create log dir %q: %w", filepath.Dir(path), err)
		}

		var err error
		file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("open log file %q: %w", path, err)
		}

		writer = io.MultiWriter(os.Stderr, file)
	}

	// Build logger with correct Options fields
	logger := log.NewWithOptions(writer, log.Options{
		Level:           level,
		Prefix:          logPrefix,
		ReportTimestamp: true,
		ReportCaller:    cfg.Log.ShowSource,
		Formatter:       formatter,
	})

	// Return handle with appropriate closer (no-op if no file)
	closer := func() error {
		if file != nil {
			return file.Close()
		}
		return nil
	}

	return &LoggerHandle{
		Logger: logger,
		Closer: closer,
	}, nil
}

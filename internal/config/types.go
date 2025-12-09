// internal/config/types.go
package config

import (
	"errors"

	"github.com/charmbracelet/log"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type LogFormat string

const (
	LogFormatText   LogFormat = "text"
	LogFormatJSON   LogFormat = "json"
	LogFormatLogfmt LogFormat = "logfmt"
)

type LogConfig struct {
	Level       LogLevel  `yaml:"level"`
	Format      LogFormat `yaml:"format"`
	ShowSource  bool      `yaml:"show_source"`
	FileEnabled bool      `yaml:"file_enabled"`
	FilePath    string    `yaml:"file_path"`
}

type PkgmgrConfig struct {
	MaxConcurrentDownloads int `yaml:"max_concurrent_downloads"` // Default: 5
}

type Config struct {
	Log    LogConfig    `yaml:"log"`
	Pkgmgr PkgmgrConfig `yaml:"pkgmgr"`
}

type LoggerHandle struct {
	Logger *log.Logger
	Closer func() error // nil if no file to close
}

var (
	ErrInvalidLogLevel               = errors.New("invalid log level")
	ErrInvalidLogFormat              = errors.New("invalid log format")
	ErrInvalidMaxConcurrentDownloads = errors.New("invalid max concurrent downloads")
)

// Validation helpers - single source of truth
func ValidLogLevels() []LogLevel {
	return []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}
}

func IsValidLogLevel(level LogLevel) bool {
	switch level {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
		return true
	default:
		return false
	}
}

func ValidLogFormats() []LogFormat {
	return []LogFormat{LogFormatText, LogFormatJSON, LogFormatLogfmt}
}

func IsValidLogFormat(format LogFormat) bool {
	switch format {
	case LogFormatText, LogFormatJSON, LogFormatLogfmt:
		return true
	default:
		return false
	}
}

func ValidMaxConcurrentDownloads(n int) bool {
	return n >= 1 && n <= 50 // Min 1, max 50 to prevent abuse
}

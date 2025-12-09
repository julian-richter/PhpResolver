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

type Config struct {
	Log LogConfig `yaml:"log"`
}

type LoggerHandle struct {
	Logger *log.Logger
	Closer func() error // nil if no file to close
}

var (
	ErrInvalidLogLevel  = errors.New("invalid log level")
	ErrInvalidLogFormat = errors.New("invalid log format")
)

func ValidLogLevels() []string {
	return []string{string(LogLevelDebug), string(LogLevelInfo),
		string(LogLevelWarn), string(LogLevelError)}
}

func IsValidLogLevel(level string) bool {
	return level == string(LogLevelDebug) ||
		level == string(LogLevelInfo) ||
		level == string(LogLevelWarn) ||
		level == string(LogLevelError)
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

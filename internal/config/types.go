package config

type LogFormat string

const (
	LogFormatText   LogFormat = "text"
	LogFormatJSON   LogFormat = "json"
	LogFormatLogfmt LogFormat = "logfmt"
)

type LogConfig struct {
	Level       string    `yaml:"level"`
	Format      LogFormat `yaml:"format"`
	ShowSource  bool      `yaml:"show_source"`
	FileEnabled bool      `yaml:"file_enabled"`
	FilePath    string    `yaml:"file_path"`
}

type Config struct {
	Log LogConfig `yaml:"log"`
}

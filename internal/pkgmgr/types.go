package pkgmgr

import (
	"encoding/json"
	"fmt"
)

// StringOrArray is a type that can unmarshal both a single string or an array of strings
type StringOrArray []string

func (s *StringOrArray) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as array first
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*s = arr
		return nil
	}

	// If that fails, try as a single string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = []string{str}
		return nil
	}

	return fmt.Errorf("value must be string or array of strings")
}

type ComposerJSON struct {
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Keywords         []string          `json:"keywords"`
	Type             string            `json:"type"`
	License          string            `json:"license"`
	Require          map[string]string `json:"require"`
	RequireDev       map[string]string `json:"require-dev,omitempty"`
	Autoload         Autoload          `json:"autoload,omitempty"`
	MinimumStability string            `json:"minimum-stability,omitempty"`
	PreferStable     bool              `json:"prefer-stable,omitempty"`
	Config           Config            `json:"config,omitempty"`
	Repositories     []Repository      `json:"repositories,omitempty"`
	AllowPlugins     map[string]bool   `json:"allow-plugins,omitempty"`
}

type Autoload struct {
	PSR4     map[string]StringOrArray `json:"psr-4,omitempty"`
	PSR0     map[string]StringOrArray `json:"psr-0,omitempty"`
	Classmap StringOrArray            `json:"classmap,omitempty"`
	Files    StringOrArray            `json:"files,omitempty"`
}

type Config struct {
	ProcessTimeout int      `json:"process-timeout,omitempty"`
	FXPAsset       FXPAsset `json:"fxp-asset,omitempty"`
}

type FXPAsset struct {
	Enabled bool `json:"enabled"`
}

type Repository struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type Package struct {
	Name     string
	Version  string
	Dist     Dist
	Autoload Autoload
}

type Dist struct {
	URL      string `json:"url"`
	Type     string `json:"type"` // zip, tar
	Checksum string `json:"checksum,omitempty"`
	Shasum   string `json:"shasum,omitempty"`
}

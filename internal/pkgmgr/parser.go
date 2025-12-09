package pkgmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

func FindComposerJSON(dir string) (string, error) {
	for {
		path := filepath.Join(dir, "composer.json")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("composer.json not found")
		}
		dir = parent
	}
}

func ParseComposerJSON(path string) (ComposerJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ComposerJSON{}, fmt.Errorf("read composer.json: %w", err)
	}

	var composer ComposerJSON
	if err := json.Unmarshal(data, &composer); err != nil {
		return ComposerJSON{}, fmt.Errorf("parse composer.json: %w", err)
	}

	return composer, nil
}

func GenerateAutoloader(autoload Autoload, vendorDir string, logger *log.Logger) error {
	// MVP: Create basic autoloader stub
	autoloadPath := filepath.Join(vendorDir, "autoload.php")

	// TODO: Real PSR-4 + files autoloading
	data := []byte(`<?php
// phpResolver autoloader stub - MVP
// TODO: Implement PSR-4, PSR-0, classmap, files autoloading
echo "phpResolver autoloader loaded\\n";
`)

	return os.WriteFile(autoloadPath, data, 0o644)
}

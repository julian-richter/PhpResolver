package pkgmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

func GenerateAutoloader(ctx context.Context, autoload Autoload, vendorDir string, logger *log.Logger) error {
	logger.Info("Generating autoloader (MVP)", "psr4_count", len(autoload.PSR4), "psr0_count", len(autoload.PSR0), "classmap_count", len(autoload.Classmap), "files_count", len(autoload.Files))

	// MVP: Create basic autoloader stub with configuration info
	autoloadPath := filepath.Join(vendorDir, "autoload.php")

	// Generate PHP content that shows what we detected
	var phpContent strings.Builder
	phpContent.WriteString(`<?php
// phpResolver autoloader stub - MVP
// This is a minimal implementation that shows detected autoload configuration
//
// DETECTED CONFIGURATION:
//`)

	if len(autoload.PSR4) > 0 {
		phpContent.WriteString(fmt.Sprintf("// PSR-4 mappings: %d\n", len(autoload.PSR4)))
		for namespace, paths := range autoload.PSR4 {
			phpContent.WriteString(fmt.Sprintf("//   %s => %v\n", namespace, paths))
		}
	}

	if len(autoload.PSR0) > 0 {
		phpContent.WriteString(fmt.Sprintf("// PSR-0 mappings: %d\n", len(autoload.PSR0)))
		for namespace, paths := range autoload.PSR0 {
			phpContent.WriteString(fmt.Sprintf("//   %s => %v\n", namespace, paths))
		}
	}

	if len(autoload.Classmap) > 0 {
		phpContent.WriteString(fmt.Sprintf("// Classmap entries: %d\n", len(autoload.Classmap)))
		for _, path := range autoload.Classmap {
			phpContent.WriteString(fmt.Sprintf("//   %s\n", path))
		}
	}

	if len(autoload.Files) > 0 {
		phpContent.WriteString(fmt.Sprintf("// Files to include: %d\n", len(autoload.Files)))
		for _, file := range autoload.Files {
			phpContent.WriteString(fmt.Sprintf("//   %s\n", file))
		}
	}

	phpContent.WriteString(`//
// TODO: Implement actual PSR-4, PSR-0, classmap, and files autoloading
echo "phpResolver autoloader loaded (MVP - configuration detected but not implemented)\n";
`)

	// Check for cancellation before file I/O
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	data := []byte(phpContent.String())
	return os.WriteFile(autoloadPath, data, 0o644)
}

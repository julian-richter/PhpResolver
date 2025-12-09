package pkgmgr

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/julian-richter/PhpResolver/internal/config"
)

func RunDumpAutoload(_ *log.Logger, _ config.Config) error {
	composerPath, err := FindComposerJSON(".")
	if err != nil {
		return fmt.Errorf("find composer.json: %w", err)
	}

	composer, err := ParseComposerJSON(composerPath)
	if err != nil {
		return fmt.Errorf("parse composer.json: %w", err)
	}

	vendorDir := filepath.Join(filepath.Dir(composerPath), "vendor")
	if err := GenerateAutoloader(composer.Autoload, vendorDir, nil); err != nil {
		return fmt.Errorf("generate autoloader: %w", err)
	}

	return nil
}

package pkgmgr

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/julian-richter/PhpResolver/internal/config"
)

func RunDumpAutoload(ctx context.Context, logger *log.Logger, cfg config.Config) error {
	composerPath, err := FindComposerJSON(".")
	if err != nil {
		return fmt.Errorf("find composer.json: %w", err)
	}

	composer, err := ParseComposerJSON(composerPath)
	if err != nil {
		return fmt.Errorf("parse composer.json: %w", err)
	}

	vendorDir := filepath.Join(filepath.Dir(composerPath), "vendor")
	logger.Info("Generating autoloader", "vendor_dir", vendorDir)
	if err := GenerateAutoloader(composer.Autoload, vendorDir, logger); err != nil {
		return fmt.Errorf("generate autoloader: %w", err)
	}

	logger.Info("Autoloader generated successfully")
	return nil
}

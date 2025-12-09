package pkgmgr

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/julian-richter/PhpResolver/internal/config"
)

// RunDumpAutoload generates the composer autoloader. Unlike RunInstall/RunUpdate which
// perform network operations requiring concurrency limits and cancellation, this function
// operates synchronously on local files. The cfg parameter is accepted for API consistency
// but currently unused since autoloader generation has no configurable behavior.
// Context is respected for cancellation consistency with other operations.
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

	// Check for cancellation before the potentially slow autoloader generation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if err := GenerateAutoloader(ctx, composer.Autoload, vendorDir, logger); err != nil {
		return fmt.Errorf("generate autoloader: %w", err)
	}

	logger.Info("Autoloader generated successfully")
	return nil
}

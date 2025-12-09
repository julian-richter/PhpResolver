// internal/pkgmgr/install.go
package pkgmgr

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/julian-richter/PhpResolver/internal/config"
)

func RunInstall(ctx context.Context, logger *log.Logger, cfg config.Config) error {
	composerPath, err := FindComposerJSON(".")
	if err != nil {
		return fmt.Errorf("find composer.json: %w", err)
	}
	logger.Info("Found composer.json", "path", composerPath)

	composer, err := ParseComposerJSON(composerPath)
	if err != nil {
		return fmt.Errorf("parse composer.json: %w", err)
	}

	vendorDir := filepath.Join(filepath.Dir(composerPath), "vendor")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		return fmt.Errorf("create vendor dir: %w", err)
	}

	// Create cache dir
	cacheDir := filepath.Join(os.Getenv("HOME"), ".phpResolver", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	// Resolve packages from custom repositories and Packagist
	packages, err := ResolvePackagesWithRepos(composer.Require, composer.Repositories, logger)
	if err != nil {
		return fmt.Errorf("resolve packages: %w", err)
	}

	// Download with configurable concurrency
	if err := DownloadPackages(ctx, packages, cacheDir, logger, cfg); err != nil {
		return fmt.Errorf("download packages: %w", err)
	}

	// Extract packages from cache to vendor/
	if err := ExtractPackages(packages, cacheDir, vendorDir, logger); err != nil {
		return fmt.Errorf("extract packages: %w", err)
	}

	if err := GenerateAutoloader(composer.Autoload, vendorDir, logger); err != nil {
		return fmt.Errorf("generate autoloader: %w", err)
	}

	logger.Info("Installation complete", "vendor_dir", vendorDir)
	return nil
}

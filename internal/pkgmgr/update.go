package pkgmgr

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/julian-richter/PhpResolver/internal/config"
)

// RunUpdate performs dependency resolution to find newer compatible versions
// and updates the installation accordingly. Currently implements basic update
// semantics without lockfile management. Without lockfile support, this is
// functionally identical to RunInstall - both resolve to latest compatible versions.
// TODO: Add composer.lock reading/writing to differentiate update from install.
func RunUpdate(ctx context.Context, logger *log.Logger, cfg config.Config) error {
	logger.Info("Starting dependency update (MVP - no lockfile support, resolves latest like install)")

	// Find and parse composer.json
	composerPath, err := FindComposerJSON(".")
	if err != nil {
		return fmt.Errorf("find composer.json: %w", err)
	}
	logger.Info("Found composer.json", "path", composerPath)

	composer, err := ParseComposerJSON(composerPath)
	if err != nil {
		return fmt.Errorf("parse composer.json: %w", err)
	}

	// Create vendor directory
	vendorDir := filepath.Join(filepath.Dir(composerPath), "vendor")
	if err := os.MkdirAll(vendorDir, 0o755); err != nil {
		return fmt.Errorf("create vendor dir: %w", err)
	}

	// Create cache dir
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home dir: %w", err)
	}
	cacheDir := filepath.Join(home, ".phpResolver", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	// TODO: Read existing composer.lock if present
	// TODO: Compare current resolutions with lockfile to detect changes

	// Re-resolve dependencies - for update, we want latest compatible versions
	// (In future, this will ignore lockfile constraints and resolve fresh)
	packages, err := ResolvePackagesWithRepos(ctx, composer.Require, composer.Repositories, logger)
	if err != nil {
		return fmt.Errorf("resolve packages: %w", err)
	}

	// TODO: Write updated composer.lock with resolved versions
	// TODO: Handle version constraint conflicts and user preferences

	// Download with configurable concurrency
	if err := DownloadPackages(ctx, packages, cacheDir, logger, cfg); err != nil {
		return fmt.Errorf("download packages: %w", err)
	}

	// Extract packages from cache to vendor/
	if err := ExtractPackages(ctx, packages, cacheDir, vendorDir, logger); err != nil {
		return fmt.Errorf("extract packages: %w", err)
	}

	if err := GenerateAutoloader(ctx, composer.Autoload, vendorDir, logger); err != nil {
		return fmt.Errorf("generate autoloader: %w", err)
	}

	logger.Info("Update complete", "vendor_dir", vendorDir)
	return nil
}

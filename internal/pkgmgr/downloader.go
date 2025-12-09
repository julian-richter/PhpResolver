package pkgmgr

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/julian-richter/PhpResolver/internal/config"
)

func DownloadPackages(ctx context.Context, packages []Package, cacheDir string, logger *log.Logger, cfg config.Config) error {
	// Create a cancellable context to stop all downloads on first error
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sem := make(chan struct{}, cfg.Pkgmgr.MaxConcurrentDownloads)
	var wg sync.WaitGroup
	errCh := make(chan error, len(packages))

	for _, pkg := range packages {
		wg.Add(1)
		go func(pkg Package) {
			defer wg.Done()

			// Try to acquire semaphore, and allow cancellation
			select {
			case sem <- struct{}{}: // Acquired semaphore
				defer func() { <-sem }() // Release semaphore
			case <-ctx.Done():
				return // Context cancelled, exit without acquiring semaphore
			}

			if err := downloadPackage(ctx, pkg, cacheDir, logger); err != nil {
				select {
				case errCh <- fmt.Errorf("package %s: %w", pkg.Name, err):
				case <-ctx.Done():
				}
			}
		}(pkg)
	}

	// Wait for completion or cancellation
	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		if err != nil {
			cancel()   // Cancel remaining downloads
			return err // Return first error
		}
	}

	logger.Info("All packages downloaded", "count", len(packages))
	return nil
}

func downloadPackage(ctx context.Context, pkg Package, cacheDir string, logger *log.Logger) error {
	cachePath := filepath.Join(cacheDir, pkg.Name, pkg.Version, fmt.Sprintf("%s.zip", pkg.Name))
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	// Skip if already exists (idempotent)
	if _, err := os.Stat(cachePath); err == nil {
		logger.Debug("Package already cached", "path", cachePath)
		return nil
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", pkg.Dist.URL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", pkg.Dist.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, pkg.Dist.URL)
	}

	// Create temp file in same directory as cache file
	tempFile, err := os.CreateTemp(filepath.Dir(cachePath), fmt.Sprintf("%s.tmp", filepath.Base(pkg.Name)))
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Set appropriate permissions (readable by owner and group, writable by owner)
	if err := os.Chmod(tempPath, 0o644); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return fmt.Errorf("set temp file permissions: %w", err)
	}

	// Ensure temp file is cleaned up on error
	defer func() {
		if tempFile != nil {
			tempFile.Close()
			os.Remove(tempPath)
		}
	}()

	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}

	// Close temp file
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	tempFile = nil // Prevent cleanup

	// Atomically rename temp file to final location
	if err := os.Rename(tempPath, cachePath); err != nil {
		os.Remove(tempPath) // Clean up temp file on rename failure
		return fmt.Errorf("rename temp file to cache: %w", err)
	}

	// Verify checksum if provided
	if pkg.Dist.Checksum != "" || pkg.Dist.Shasum != "" {
		expectedHash := pkg.Dist.Checksum
		if expectedHash == "" {
			expectedHash = pkg.Dist.Shasum
		}

		// Reopen file for verification
		file, err := os.Open(cachePath)
		if err != nil {
			return fmt.Errorf("reopen cache file for verification: %w", err)
		}
		defer file.Close()

		// Compute SHA1 hash
		hasher := sha1.New()
		if _, err := io.Copy(hasher, file); err != nil {
			return fmt.Errorf("compute checksum: %w", err)
		}

		actualHash := hex.EncodeToString(hasher.Sum(nil))
		if actualHash != expectedHash {
			// Remove corrupted file
			os.Remove(cachePath)
			return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
		}
	}

	logger.Info("Downloaded", "package", pkg.Name, "version", pkg.Version, "path", cachePath)
	return nil
}

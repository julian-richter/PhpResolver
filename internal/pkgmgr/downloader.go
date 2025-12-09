package pkgmgr

import (
	"context"
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
	sem := make(chan struct{}, cfg.Pkgmgr.MaxConcurrentDownloads)
	var wg sync.WaitGroup
	errCh := make(chan error, len(packages))

	for _, pkg := range packages {
		wg.Add(1)
		go func(pkg Package) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore (signals)
			defer func() { <-sem }() // Release semaphore (signals)

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

	file, err := os.Create(cachePath)
	if err != nil {
		return fmt.Errorf("create cache file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	logger.Info("Downloaded", "package", pkg.Name, "version", pkg.Version, "path", cachePath)
	return nil
}

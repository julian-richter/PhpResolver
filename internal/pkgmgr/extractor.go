package pkgmgr

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

// ExtractPackages extracts downloaded zip files to vendor directory
// following Composer's vendor/vendor-name/package-name structure
func ExtractPackages(ctx context.Context, packages []Package, cacheDir, vendorDir string, logger *log.Logger) error {
	var errors []string
	var failedPackages []string

	for _, pkg := range packages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := extractPackage(ctx, pkg, cacheDir, vendorDir, logger); err != nil {
			logger.Error("Failed to extract package", "package", pkg.Name, "error", err)
			errors = append(errors, fmt.Sprintf("%s: %v", pkg.Name, err))
			failedPackages = append(failedPackages, pkg.Name)
			continue // Continue with remaining packages
		}
	}

	// Log summary of results
	successCount := len(packages) - len(failedPackages)
	if successCount > 0 {
		logger.Info("Successfully extracted packages", "success_count", successCount)
	}
	if len(failedPackages) > 0 {
		logger.Warn("Some packages failed to extract", "failed_count", len(failedPackages), "failed_packages", failedPackages)
	} else {
		logger.Info("All packages extracted", "count", len(packages))
	}

	// Return combined error if any packages failed
	if len(errors) > 0 {
		return fmt.Errorf("failed to extract %d package(s): %s", len(errors), strings.Join(errors, "; "))
	}

	return nil
}

func extractPackage(ctx context.Context, pkg Package, cacheDir, vendorDir string, logger *log.Logger) error {
	// Build cache path: ~/.phpResolver/cache/vendor/package/version/vendor-package.zip
	cachePath := filepath.Join(cacheDir, pkg.Name, pkg.Version, fmt.Sprintf("%s.zip", pkg.Name))

	// Build vendor path: vendor/vendor-name/package-name/
	vendorPath := filepath.Join(vendorDir, pkg.Name)

	// Ensure parent directory exists before creating temp directory
	parentDir := filepath.Dir(vendorPath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("create parent dir %s: %w", parentDir, err)
	}

	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp(parentDir, filepath.Base(vendorPath)+".tmp")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		// Clean up temp directory on any error
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
	}()

	// Open zip file
	zipReader, err := zip.OpenReader(cachePath)
	if err != nil {
		return fmt.Errorf("open zip file %s: %w", cachePath, err)
	}
	defer zipReader.Close()

	// Extract files
	// Composer zip files typically have a root directory with the package name
	// We need to strip that root directory when extracting
	rootDir := computeCommonPrefix(zipReader.File)

	for _, file := range zipReader.File {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := extractZipFile(file, tempDir, rootDir, logger); err != nil {
			return fmt.Errorf("extract file %s: %w", file.Name, err)
		}
	}

	// Perform atomic directory swap to avoid data loss
	backupPath := vendorPath + ".backup"

	// Create backup of existing vendor directory (if it exists)
	if _, err := os.Stat(vendorPath); err == nil {
		// Vendor directory exists, create backup
		if err := os.Rename(vendorPath, backupPath); err != nil {
			return fmt.Errorf("create backup of existing vendor dir: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check existing vendor dir: %w", err)
	}

	// Attempt to move temp directory to final location
	if err := os.Rename(tempDir, vendorPath); err != nil {
		// Rename failed, attempt to restore from backup
		if _, err := os.Stat(backupPath); err == nil {
			if restoreErr := os.Rename(backupPath, vendorPath); restoreErr != nil {
				logger.Error("Failed to restore from backup after swap failure",
					"package", pkg.Name, "backup_path", backupPath, "error", restoreErr)
			}
		}
		return fmt.Errorf("move temp dir to vendor (attempted restore from backup): %w", err)
	}

	// Swap successful, clean up backup if it exists
	if _, err := os.Stat(backupPath); err == nil {
		if err := os.RemoveAll(backupPath); err != nil {
			logger.Warn("Failed to clean up backup directory", "backup_path", backupPath, "error", err)
			// Don't return error for cleanup failure - the main operation succeeded
		}
	}

	// Extraction and swap succeeded, don't clean up temp directory (it's now vendorPath)
	tempDir = ""

	logger.Info("Extracted package", "package", pkg.Name, "version", pkg.Version, "to", vendorPath)
	return nil
}

func extractZipFile(file *zip.File, destDir, stripPrefix string, logger *log.Logger) error {
	// Get the file path relative to strip prefix
	relativePath := file.Name
	if stripPrefix != "" && strings.HasPrefix(relativePath, stripPrefix) {
		relativePath = strings.TrimPrefix(relativePath, stripPrefix)
	}

	// Skip empty paths (root directory itself)
	if relativePath == "" {
		return nil
	}

	// Build destination path
	destPath := filepath.Join(destDir, relativePath)

	// Prevent zip slip vulnerability
	if !strings.HasPrefix(destPath, filepath.Clean(destDir)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path: %s", destPath)
	}

	// Check if it's a directory
	if file.FileInfo().IsDir() {
		return os.MkdirAll(destPath, file.Mode())
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}

	// Extract file
	srcFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("open file in zip: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return fmt.Errorf("create dest file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("copy file contents: %w", err)
	}

	return nil
}

// computeCommonPrefix finds the common directory prefix across all zip entries
func computeCommonPrefix(files []*zip.File) string {
	if len(files) == 0 {
		return ""
	}

	// Find first non-empty file name
	var firstComponents []string
	for _, file := range files {
		if file.Name != "" {
			firstComponents = strings.Split(strings.TrimSuffix(file.Name, "/"), "/")
			break
		}
	}

	if len(firstComponents) == 0 {
		return ""
	}

	// Initialize commonComponents from first file
	commonComponents := make([]string, len(firstComponents))
	copy(commonComponents, firstComponents)

	// For each subsequent file, truncate commonComponents in-place
	for _, file := range files {
		if file.Name == "" {
			continue
		}

		components := strings.Split(strings.TrimSuffix(file.Name, "/"), "/")

		// Find the common prefix length
		minLen := len(commonComponents)
		if len(components) < minLen {
			minLen = len(components)
		}

		// Truncate at first difference
		for i := 0; i < minLen; i++ {
			if commonComponents[i] != components[i] {
				commonComponents = commonComponents[:i]
				break
			}
		}

		// If we found a difference at position 0, no common prefix
		if len(commonComponents) == 0 {
			break
		}
	}

	// Return joined components with trailing slash if we have a common prefix
	if len(commonComponents) > 0 {
		return strings.Join(commonComponents, "/") + "/"
	}

	return ""
}

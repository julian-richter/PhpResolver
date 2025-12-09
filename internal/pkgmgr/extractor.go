package pkgmgr

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

// ExtractPackages extracts downloaded zip files to vendor directory
// following Composer's vendor/vendor-name/package-name structure
func ExtractPackages(packages []Package, cacheDir, vendorDir string, logger *log.Logger) error {
	for _, pkg := range packages {
		if err := extractPackage(pkg, cacheDir, vendorDir, logger); err != nil {
			logger.Error("Failed to extract package", "package", pkg.Name, "error", err)
			return fmt.Errorf("extract %s: %w", pkg.Name, err)
		}
	}
	logger.Info("All packages extracted", "count", len(packages))
	return nil
}

func extractPackage(pkg Package, cacheDir, vendorDir string, logger *log.Logger) error {
	// Build cache path: ~/.phpResolver/cache/vendor/package/version/vendor-package.zip
	cachePath := filepath.Join(cacheDir, pkg.Name, pkg.Version, fmt.Sprintf("%s.zip", pkg.Name))

	// Check if zip file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return fmt.Errorf("zip file not found: %s", cachePath)
	}

	// Build vendor path: vendor/vendor-name/package-name/
	vendorPath := filepath.Join(vendorDir, pkg.Name)

	// Create vendor directory
	if err := os.MkdirAll(vendorPath, 0o755); err != nil {
		return fmt.Errorf("create vendor dir: %w", err)
	}

	// Open zip file
	zipReader, err := zip.OpenReader(cachePath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zipReader.Close()

	// Extract files
	// Composer zip files typically have a root directory with the package name
	// We need to strip that root directory when extracting
	var rootDir string
	if len(zipReader.File) > 0 {
		// Detect root directory from first file
		firstPath := zipReader.File[0].Name
		if idx := strings.Index(firstPath, "/"); idx != -1 {
			rootDir = firstPath[:idx+1]
		}
	}

	for _, file := range zipReader.File {
		if err := extractZipFile(file, vendorPath, rootDir, logger); err != nil {
			return fmt.Errorf("extract file %s: %w", file.Name, err)
		}
	}

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

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
	for _, pkg := range packages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := extractPackage(ctx, pkg, cacheDir, vendorDir, logger); err != nil {
			logger.Error("Failed to extract package", "package", pkg.Name, "error", err)
			return fmt.Errorf("extract %s: %w", pkg.Name, err)
		}
	}
	logger.Info("All packages extracted", "count", len(packages))
	return nil
}

func extractPackage(ctx context.Context, pkg Package, cacheDir, vendorDir string, logger *log.Logger) error {
	// Build cache path: ~/.phpResolver/cache/vendor/package/version/vendor-package.zip
	cachePath := filepath.Join(cacheDir, pkg.Name, pkg.Version, fmt.Sprintf("%s.zip", pkg.Name))

	// Check if zip file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return fmt.Errorf("zip file not found: %s", cachePath)
	}

	// Build vendor path: vendor/vendor-name/package-name/
	vendorPath := filepath.Join(vendorDir, pkg.Name)

	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp(filepath.Dir(vendorPath), filepath.Base(vendorPath)+".tmp")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer func() {
		// Clean up temp directory on any error
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
	}()

	// Remove any pre-existing vendor directory to avoid conflicts
	if err := os.RemoveAll(vendorPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove existing vendor dir: %w", err)
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

	// Atomically move temp directory to final location
	if err := os.Rename(tempDir, vendorPath); err != nil {
		return fmt.Errorf("move temp dir to vendor: %w", err)
	}

	// Extraction succeeded, don't clean up temp directory
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

	var paths []string
	for _, file := range files {
		if file.Name != "" {
			paths = append(paths, file.Name)
		}
	}

	if len(paths) == 0 {
		return ""
	}

	// Split all paths into components
	var pathComponents [][]string
	for _, path := range paths {
		components := strings.Split(strings.TrimSuffix(path, "/"), "/")
		pathComponents = append(pathComponents, components)
	}

	// Find common prefix components
	minLen := len(pathComponents[0])
	for _, components := range pathComponents {
		if len(components) < minLen {
			minLen = len(components)
		}
	}

	var commonComponents []string
	for i := 0; i < minLen; i++ {
		component := pathComponents[0][i]
		isCommon := true

		for _, components := range pathComponents[1:] {
			if components[i] != component {
				isCommon = false
				break
			}
		}

		if !isCommon {
			break
		}
		commonComponents = append(commonComponents, component)
	}

	// Join components and add trailing slash if we have a common prefix
	if len(commonComponents) > 0 {
		return strings.Join(commonComponents, "/") + "/"
	}

	return ""
}

package pkgmgr

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
)

var (
	phpExtRE     = regexp.MustCompile(`^ext-(\w+)$`)
	npmAssetRE   = regexp.MustCompile(`^npm-asset/`)
	bowerAssetRE = regexp.MustCompile(`^bower-asset/`)
)

func ResolvePackages(require map[string]string, logger *log.Logger) ([]Package, error) {
	return ResolvePackagesWithRepos(require, nil, logger)
}

func ResolvePackagesWithRepos(require map[string]string, repositories []Repository, logger *log.Logger) ([]Package, error) {
	var packages []Package
	var errors []string

	for name, constraint := range require {
		// Skip PHP/platform requirements for MVP
		if isPlatformRequirement(name) {
			logger.Debug("Skipping platform requirement", "package", name)
			continue
		}

		pkg, err := resolvePackage(name, constraint, repositories, logger)
		if err != nil {
			logger.Warn("Failed to resolve package (skipping)", "package", name, "error", err.Error())
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			continue // Skip this package but continue with others
		}
		packages = append(packages, pkg)
	}

	// Log summary
	if len(errors) > 0 {
		logger.Warn("Some packages could not be resolved", "count", len(errors), "total", len(require))
	}
	logger.Info("Package resolution complete", "resolved", len(packages), "failed", len(errors))

	return packages, nil
}

func resolvePackage(name, constraint string, repositories []Repository, logger *log.Logger) (Package, error) {
	// Check if this is an asset package (npm-asset/ or bower-asset/)
	isAsset := npmAssetRE.MatchString(name) || bowerAssetRE.MatchString(name)

	if isAsset {
		logger.Debug("Detected asset package", "package", name)
		// Asset packages must be resolved from asset-packagist.org
		// Check if asset-packagist is in the repositories list
		for _, repo := range repositories {
			if repo.Type == "composer" && strings.Contains(repo.URL, "asset-packagist.org") {
				logger.Debug("Trying asset-packagist", "package", name, "url", repo.URL)
				pkg, err := queryComposerRepository(repo.URL, name, constraint, logger)
				if err == nil {
					return pkg, nil
				}
				logger.Debug("Asset package not found in asset-packagist", "package", name, "error", err)
			}
		}
		// If asset-packagist is not configured or package not found, return an error
		return Package{}, fmt.Errorf("asset package %s not found in asset-packagist.org", name)
	}

	// Try custom composer repositories first (skip asset-packagist as it was tried above for assets)
	for _, repo := range repositories {
		if repo.Type == "composer" && !strings.Contains(repo.URL, "asset-packagist.org") {
			logger.Debug("Trying custom composer repository", "package", name, "repo", repo.URL)
			pkg, err := queryComposerRepository(repo.URL, name, constraint, logger)
			if err == nil {
				return pkg, nil
			}
			logger.Debug("Package not found in custom repository", "package", name, "repo", repo.URL, "error", err)
		} else if repo.Type == "git" {
			// Git repositories require special handling
			// For now, just log once per package and skip
			logger.Debug("Skipping git repository (not yet implemented)", "package", name, "repo", repo.URL)
		}
	}

	// Fallback to Packagist
	logger.Debug("Trying packagist.org", "package", name)
	return queryComposerRepository("https://packagist.org", name, constraint, logger)
}

func queryComposerRepository(baseURL, name, constraint string, logger *log.Logger) (Package, error) {
	url := fmt.Sprintf("%s/packages/%s.json", baseURL, name)
	resp, err := http.Get(url)
	if err != nil {
		return Package{}, fmt.Errorf("repository lookup %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Package{}, fmt.Errorf("repository %s returned %s for %s", baseURL, resp.Status, name)
	}

	var data struct {
		Package struct {
			Versions map[string]struct {
				Dist Dist `json:"dist"`
			} `json:"versions"`
		} `json:"package"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return Package{}, fmt.Errorf("decode repository response for %s: %w", name, err)
	}

	// MVP: Pick first stable version
	for version, vdata := range data.Package.Versions {
		if vdata.Dist.URL != "" && strings.HasPrefix(vdata.Dist.URL, "https://") {
			logger.Debug("Resolved package", "package", name, "version", version, "repo", baseURL)
			return Package{
				Name:    name,
				Version: version,
				Dist:    vdata.Dist,
			}, nil
		}
	}

	return Package{}, fmt.Errorf("no HTTPS dist found for %s", name)
}

func isPlatformRequirement(name string) bool {
	return phpExtRE.MatchString(name) || name == "php"
}
func isAssetPackage(name string) bool {
	return npmAssetRE.MatchString(name) || bowerAssetRE.MatchString(name)
}

package pkgmgr

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/julian-richter/PhpResolver/internal/config"
)

func RunUpdate(ctx context.Context, logger *log.Logger, cfg config.Config) error {
	logger.Info("phpResolver update - running install")
	return RunInstall(ctx, logger, cfg)
}

// cmd/app/main.go
package main

import (
	"os"

	charmLog "github.com/charmbracelet/log"
	"github.com/julian-richter/PhpResolver/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// fall back to a minimal stderr logger
		fallback := charmLog.NewWithOptions(os.Stderr, charmLog.Options{
			Level: charmLog.ErrorLevel,
		})
		fallback.Fatal("failed to load config", "err", err)
	}

	logger, err := config.NewLogger(cfg)
	if err != nil {
		// cannot continue if logger cannot be built
		charmLog.New(os.Stderr).Fatal("failed to initialize logger", "err", err)
	}

	charmLog.SetDefault(logger)

	if err := run(logger, cfg); err != nil {
		logger.Fatal("application error", "err", err)
	}
}

func run(logger *charmLog.Logger, cfg config.Config) error {
	return nil
}

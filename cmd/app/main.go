// cmd/app/main.go
package main

import (
	"os"

	"github.com/charmbracelet/log"
	"github.com/julian-richter/PhpResolver/internal/config"
)

func createFallbackLogger() *log.Logger {
	return log.NewWithOptions(os.Stderr, log.Options{
		Level: log.ErrorLevel,
	})
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		// Fallback logger for config errors
		fallback := createFallbackLogger()
		fallback.Fatal("failed to load config", "err", err)
	}

	handle, err := config.NewLogger(cfg)
	if err != nil {
		fallback := createFallbackLogger()
		fallback.Fatal("failed to initialize logger", "err", err)
	}
	defer func() {
		if handle.Closer != nil {
			if err := handle.Closer(); err != nil {
				// Log but don't fail on close error
				log.Error("failed to close log file", "err", err)
			}
		}
	}()

	log.SetDefault(handle.Logger)

	// Inject logger into app
	if err := run(handle.Logger, cfg); err != nil {
		handle.Logger.Fatal("application error", "err", err)
	}
}

func run(logger *log.Logger, _ config.Config) error {
	logger.Infof("starting application")
	return nil
}

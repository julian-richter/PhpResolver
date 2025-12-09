// cmd/app/main.go - Manual CLI without external deps
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/julian-richter/PhpResolver/internal/config"
	"github.com/julian-richter/PhpResolver/internal/pkgmgr"
)

func createFallbackLogger() *log.Logger {
	return log.NewWithOptions(os.Stderr, log.Options{
		Level: log.ErrorLevel,
	})
}

func main() {
	cfg, err := config.Load()
	if err != nil {
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
				handle.Logger.Error("failed to close log file", "err", err)
			}
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := runCLI(ctx, os.Args, handle.Logger, cfg); err != nil {
		handle.Logger.Fatal("application error", "err", err)
	}
}

func runCLI(ctx context.Context, args []string, logger *log.Logger, cfg config.Config) error {
	if len(args) < 2 {
		printUsage(logger)
		return fmt.Errorf("no command specified")
	}

	cmd := strings.ToLower(args[1])
	switch cmd {
	case "install":
		return pkgmgr.RunInstall(ctx, logger, cfg)
	case "update":
		return pkgmgr.RunUpdate(ctx, logger, cfg)
	case "dump-autoload":
		return pkgmgr.RunDumpAutoload(ctx, logger, cfg)
	case "help", "-h", "--help":
		printUsage(logger)
		return nil
	default:
		printUsage(logger)
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

// printUsage prints help text to stdout intentionally bypassing the logger
// to avoid timestamp/JSON formatting that would make the output less readable
func printUsage(logger *log.Logger) {
	fmt.Println(`phpResolver - Drop-in Composer replacement

Usage:
  phpResolver install        Install project dependencies
  phpResolver update         Update dependencies to their newest versions  
  phpResolver dump-autoload  Dump the autoloader`)
}

package main

import (
	"context"
	"flag"
	"log/slog"
	"os/signal"
	"syscall"

	"mc-daemon/internal/config"
	"mc-daemon/internal/config/installer"

	"os"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// Cleanup signal notification resources when main exits
	defer stop()

	// Setup Logger Handler options
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				// Reformatting time to HH:MM:SS
				t := a.Value.Time()
				return slog.String(slog.TimeKey, t.Format("15:04:05"))
			}
			return a
		},
	}

	// Initialize root logger
	rootLogger := slog.New(slog.NewTextHandler(os.Stdout, opts))

	// Setup & Parse command line args
	configPath := parseFlags()

	cfg, err := config.Load(configPath, rootLogger)
	if err != nil {
		rootLogger.Error("fatal initialization failure", "error", err)
		os.Exit(1)
	}

	rootLogger.Info("Loaded Configuration", "config", cfg)

	installer, err := installer.New(&cfg, rootLogger)
	if err != nil {
		rootLogger.Error("fatal initialization failure", "error", err)
	}

	// Test URL (Vanilla Minecraft server JAR)
	downloadUrl := "https://piston-data.mojang.com/v1/objects/823e2250d24b3ddac457a60c92a6a941943fcd6a/server.jar"

	if err := installer.EnsureInstalled(ctx, downloadUrl); err != nil {
		rootLogger.Error("Unble to ensure installation", "error", err)
	}

	rootLogger.Info("server binary setup complete")

}

func parseFlags() string {
	configPath := flag.String("config", "", "path to the config file")
	flag.Parse()
	return *configPath
}

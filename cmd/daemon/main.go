package main

import (
	"flag"
	"log/slog"

	"mc-daemon/internal/config"

	"os"
)

func main() {
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
}

func parseFlags() string {
	configPath := flag.String("config", "", "path to the config file")
	flag.Parse()
	return *configPath
}

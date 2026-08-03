package installer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"mc-daemon/internal/config"
)

var (
	ErrNilConfig      = errors.New("installer: config cannot be nil")
	ErrDownloadFailed = errors.New("installer: failed to download server binary")
	ErrHashMismatch   = errors.New("installer: checksum verification failed")
)

// Installer manages verifiying and fetching server binaries
type Installer struct {
	logger *slog.Logger
	cfg    *config.Config
}

// Creates a new installer instance bound to a configuration and logger
func New(cfg *config.Config, logger *slog.Logger) (*Installer, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	if logger == nil {
		logger = slog.Default()
	}

	return &Installer{
		cfg:    cfg,
		logger: logger.With("component", "installer"),
	}, nil
}

func (i *Installer) EnsureInstalled(ctx context.Context, downloadUrl string) error {
	installDir := &i.cfg.Paths.InstallDir
	jarFile := &i.cfg.Paths.JarFile

	targetPath := filepath.Join(*installDir, *jarFile)

	// Ensure destiation directory exists
	if err := os.MkdirAll(*installDir, 0755); err != nil {
		return fmt.Errorf("installer: failed to create target directory: %w", err)
	}

	i.logger.Info("checking server binary status", "path", targetPath)

	// Determine if server file already exists on disk
	if i.fileExists(targetPath) {
		i.logger.Info("server binary is present", "path", targetPath)
		return nil
	}

	if len(downloadUrl) == 0 {
		return fmt.Errorf("%w: download URL cannot be empty", ErrDownloadFailed)
	}

	i.logger.Info("downloading server binary",
		"type", i.cfg.Server.Type,
		"version", i.cfg.Server.Version,
		"url", downloadUrl,
		"target_path", targetPath,
	)

	if err := i.download(ctx, targetPath, downloadUrl); err != nil {
		return err
	}

	i.logger.Info("server binary successfully installed", "path", targetPath)
	return nil
}

func (i *Installer) fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

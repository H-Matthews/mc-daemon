package installer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// download fetches the server binary from a remote URL
// and writes it atomically to targetPath
func (i *Installer) download(ctx context.Context, targetPath string, downloadURL string) error {
	if downloadURL == "" {
		return errors.New("installer: download URL is empty")
	}

	// Prepare HTTP request with cancellation/timeout context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("installer: failed to create HTTP request: %w", err)
	}

	// Identify client cleanly to remote CDNs
	req.Header.Set("User-Agent", "Minecraft-Daemon-Installer/0.1")

	// Execute HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return errors.New("installer: download canceled by context")
		}

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return errors.New("installer: download timed out")
		}

		return fmt.Errorf("installer: network request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("installer: server returned HTTP status %d", resp.StatusCode)
	}

	// Create temp file in target dir for atomic write
	targetDir := filepath.Dir(targetPath)
	tempFile, err := os.CreateTemp(targetDir, ".server-*.tmp")
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("installer: permission denied creating temporary file in %s", targetDir)
		}
	}
	tempPath := tempFile.Name()

	// Ensure cleanup if anything fails before the atomic rename
	downloadSuccess := false
	defer func() {
		tempFile.Close()
		if !downloadSuccess {
			// Cleanup partial download
			_ = os.Remove(tempPath)
		}
	}()

	// Stream HTTP response body directly to temporary file
	i.logger.Info("streaming server binary download...", "url", downloadURL, "temp_path", tempPath)

	bytesWritten, err := io.Copy(tempFile, resp.Body)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return errors.New("installer: download interrupted")
		}
		return fmt.Errorf("installer: failed while writing download stream: %w", err)
	}

	i.logger.Info("download completed", "bytes", bytesWritten)

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("installer: failed to flush file to disk: %w", err)
	}

	// Swap temp with actual target
	if err := os.Rename(tempPath, targetPath); err != nil {
		return fmt.Errorf("installer: failed to move download file into place: %w", err)
	}

	return nil
}

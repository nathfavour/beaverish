package daemon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nathfavour/beaverish/pkg/logger"
)

// ConfigDir returns $XDG_CONFIG_HOME/beaverish or ~/.config/beaverish
func ConfigDir() string {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			xdg = filepath.Join(home, ".config")
		}
	}
	if xdg == "" {
		return "config"
	}
	dir := filepath.Join(xdg, "beaverish")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

// UpdateFilePath returns the path to trigger dynamic live reloads
func UpdateFilePath() string {
	return filepath.Join(ConfigDir(), "update")
}

// WatchForUpdates watches the config directory for "update" file touches or config modifications.
// When detected, it gracefully restarts the binary with the same arguments using syscall.Exec.
func WatchForUpdates(ctx context.Context, onReload func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Warnf("Unable to initialize file watcher: %v", err)
		return
	}

	configDir := ConfigDir()
	updateFile := UpdateFilePath()

	// Ensure update file exists so it can be watched/touched
	if _, err := os.Stat(updateFile); os.IsNotExist(err) {
		_ = os.WriteFile(updateFile, []byte("ready\n"), 0644)
	}

	if err := watcher.Add(configDir); err != nil {
		logger.Warnf("Could not watch config dir %s: %v", configDir, err)
		return
	}

	go func() {
		defer watcher.Close()
		debounceTimer := time.NewTimer(0)
		if !debounceTimer.Stop() {
			<-debounceTimer.C
		}

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Detect update trigger or config changes
				base := filepath.Base(event.Name)
				if base == "update" || base == "config.json" {
					if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Chmod) != 0 {
						logger.Infof("🔔 Detected change in %s! Triggering seamless live reload...", base)
						if onReload != nil {
							onReload()
						}
						// Restart process seamlessly
						restartProcess()
						return
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Warnf("File watcher error: %v", err)
			}
		}
	}()
}

func restartProcess() {
	binary, err := exec.LookPath(os.Args[0])
	if err != nil {
		binary = os.Args[0]
	}

	logger.Infof("♻️ Re-executing Beaverish process: %s", binary)
	// Give a tiny moment for listeners to flush
	time.Sleep(150 * time.Millisecond)

	// Replace process in-place with new binary
	err = syscall.Exec(binary, os.Args, os.Environ())
	if err != nil {
		logger.Errorf("Failed to exec restart: %v", err)
	}
}

// TouchUpdate triggers a reload signal across any running instance
func TouchUpdate() error {
	f := UpdateFilePath()
	return os.WriteFile(f, []byte(time.Now().Format(time.RFC3339)+"\n"), 0644)
}

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

// WatchForUpdates watches the config directory for "update" file touches or binary replacements.
// When detected, it gracefully restarts the binary with the same arguments using syscall.Exec.
func WatchForUpdates(ctx context.Context, onReload func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Warnf("Unable to initialize file watcher: %v", err)
		return
	}

	configDir := ConfigDir()
	updateFile := UpdateFilePath()

	if _, err := os.Stat(updateFile); os.IsNotExist(err) {
		_ = os.WriteFile(updateFile, []byte("ready\n"), 0644)
	}

	if err := watcher.Add(configDir); err != nil {
		logger.Warnf("Could not watch config dir %s: %v", configDir, err)
		return
	}

	// Also watch the executable's directory to catch binary updates (e.g. from anyisland update)
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		_ = watcher.Add(execDir)
		logger.Debugf("Watching binary dir: %s", execDir)
	}

	go func() {
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				base := filepath.Base(event.Name)
				execName := filepath.Base(execPath)

				isUpdateTrigger := base == "update"
				isConfigTrigger := base == "config.json"
				isBinaryTrigger := base == execName || base == "beaverish"

				if isUpdateTrigger || isConfigTrigger || isBinaryTrigger {
					if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Chmod|fsnotify.Rename) != 0 {
						logger.Infof("🔔 Change detected in %s! Initiating automatic in-place restart...", base)
						if onReload != nil {
							onReload()
						}
						// Clean socket before re-exec so new instance binds smoothly
						CleanSocket()
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
	execPath, err := os.Executable()
	if err != nil {
		execPath, err = exec.LookPath(os.Args[0])
		if err != nil {
			execPath = os.Args[0]
		}
	}

	logger.Infof("♻️ In-Place Re-executing Beaverish: %s", execPath)
	time.Sleep(200 * time.Millisecond)

	err = syscall.Exec(execPath, os.Args, os.Environ())
	if err != nil {
		logger.Errorf("Failed to re-exec process: %v. Spawning backup process...", err)
		cmd := exec.Command(execPath, os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Start()
		os.Exit(0)
	}
}

// TouchUpdate triggers a reload signal across any running instance
func TouchUpdate() error {
	f := UpdateFilePath()
	return os.WriteFile(f, []byte(time.Now().Format(time.RFC3339)+"\n"), 0644)
}

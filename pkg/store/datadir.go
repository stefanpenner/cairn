package store

import (
	"os"
	"path/filepath"
	"runtime"
)

// DefaultDataDir returns the OS-appropriate default data directory for cairn.
//
//   - macOS:   ~/Library/Application Support/cairn
//   - Linux:   $XDG_DATA_HOME/cairn (fallback ~/.local/share/cairn)
//   - Windows: %LOCALAPPDATA%\cairn (fallback %APPDATA%\cairn)
func DefaultDataDir() string {
	return defaultDataDirForOS(runtime.GOOS)
}

func defaultDataDirForOS(goos string) string {
	home, _ := os.UserHomeDir()

	switch goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "cairn")
	case "windows":
		if dir := os.Getenv("LOCALAPPDATA"); dir != "" {
			return filepath.Join(dir, "cairn")
		}
		if dir := os.Getenv("APPDATA"); dir != "" {
			return filepath.Join(dir, "cairn")
		}
		return filepath.Join(home, "cairn")
	default: // linux, freebsd, etc.
		if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
			return filepath.Join(dir, "cairn")
		}
		return filepath.Join(home, ".local", "share", "cairn")
	}
}

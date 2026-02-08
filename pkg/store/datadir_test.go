package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultDataDirMacOS(t *testing.T) {
	home, _ := os.UserHomeDir()
	dir := defaultDataDirForOS("darwin")
	assert.Equal(t, filepath.Join(home, "Library", "Application Support", "cairn"), dir)
}

func TestDefaultDataDirLinux(t *testing.T) {
	home, _ := os.UserHomeDir()

	// Without XDG_DATA_HOME
	t.Setenv("XDG_DATA_HOME", "")
	dir := defaultDataDirForOS("linux")
	assert.Equal(t, filepath.Join(home, ".local", "share", "cairn"), dir)

	// With XDG_DATA_HOME
	t.Setenv("XDG_DATA_HOME", "/custom/data")
	dir = defaultDataDirForOS("linux")
	assert.Equal(t, filepath.Join("/custom/data", "cairn"), dir)
}

func TestDefaultDataDirWindows(t *testing.T) {
	// With LOCALAPPDATA
	t.Setenv("LOCALAPPDATA", `C:\Users\test\AppData\Local`)
	dir := defaultDataDirForOS("windows")
	assert.Equal(t, filepath.Join(`C:\Users\test\AppData\Local`, "cairn"), dir)

	// Without LOCALAPPDATA, with APPDATA
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("APPDATA", `C:\Users\test\AppData\Roaming`)
	dir = defaultDataDirForOS("windows")
	assert.Equal(t, filepath.Join(`C:\Users\test\AppData\Roaming`, "cairn"), dir)
}

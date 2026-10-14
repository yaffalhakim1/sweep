package app

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
)

// Settings is what the app remembers between runs. The window's own bounds
// are remembered by MyGo through the window's StateKey; only the choices
// the app itself owns live here.
type Settings struct {
	// Dark is the appearance the app was left in. A first run follows the
	// desktop instead, which is what the button then switches away from.
	Dark *bool `json:"dark,omitempty"`
	// View is the last view shown, so the app reopens the way it was left.
	View int `json:"view"`
	// Roots are the directories to scan. Empty means the user's home.
	Roots []string `json:"roots,omitempty"`
}

// settingsFile is the settings file's name inside the app's per-user data
// directory.
const settingsFile = "settings.json"

// LoadSettings reads what was remembered. A missing or unreadable file is
// not an error: it means a first run, which is what the defaults are for.
func LoadSettings() Settings {
	var s Settings
	path, err := settingsPath()
	if err != nil {
		return s
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	return s
}

// saveSettings writes the remembered choices. A failure is ignored on
// purpose: the choice still applies to this run, and losing it on the next
// start is not worth interrupting the user over.
func (a *App) saveSettings() {
	path, err := settingsPath()
	if err != nil {
		return
	}
	data, err := json.Marshal(Settings{
		Dark:  &a.theme,
		View:  a.viewIdx,
		Roots: a.roots,
	})
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, data, 0o644)
}

// settingsPath is where the settings live: the app's per-user data
// directory, so they sit beside the window state MyGo keeps.
func settingsPath() (string, error) {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, settingsFile), nil
}

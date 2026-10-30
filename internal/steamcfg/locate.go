package steamcfg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RustyRaptor/deckhud/internal/vdf"
)

// Locator finds controller config files inside a Steam installation.
type Locator struct {
	SteamRoot string // e.g. ~/.local/share/Steam
	AccountID string // optional; autodetected when there is exactly one account
}

// ErrUnsupported marks configs deckhud can't read (e.g. binary workshop files).
var ErrUnsupported = errors.New("steamcfg: unsupported config source")

// ConfigDir returns ".../Steam Controller Configs/<account>/config".
func (l Locator) ConfigDir() (string, error) {
	base := filepath.Join(l.SteamRoot, "steamapps", "common", "Steam Controller Configs")
	if l.AccountID != "" {
		return filepath.Join(base, l.AccountID, "config"), nil
	}
	matches, _ := filepath.Glob(filepath.Join(base, "*", "config"))
	if len(matches) != 1 {
		return "", fmt.Errorf("steamcfg: found %d Steam accounts under %s; set AccountID", len(matches), base)
	}
	return matches[0], nil
}

// Resolve returns the config file Steam uses for appID. configSets are
// configset file names in precedence order, as logged by Steam (the
// per-controller set first, then configset_controller_neptune.vdf).
func (l Locator) Resolve(appID string, configSets []string) (string, error) {
	dir, err := l.ConfigDir()
	if err != nil {
		return "", err
	}
	for _, name := range configSets {
		root, err := parseFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		entry := root.Get("controller_config").Get(appID)
		if entry == nil {
			continue
		}
		switch {
		case entry.Str("autosave") == "1":
			suffix := strings.TrimPrefix(name, "configset_")
			return filepath.Join(dir, appID, suffix), nil
		case entry.Str("template") != "":
			t := entry.Str("template")
			if rest, ok := strings.CutPrefix(t, "CLOUD_"); ok {
				return filepath.Join(dir, filepath.FromSlash(rest)+".vdf"), nil
			}
			return filepath.Join(l.SteamRoot, "controller_base", "templates", t), nil
		case entry.Str("workshop") != "":
			return "", fmt.Errorf("%w: workshop config %s for app %s", ErrUnsupported, entry.Str("workshop"), appID)
		}
	}
	return "", fmt.Errorf("steamcfg: no config for app %s in %v", appID, configSets)
}

func parseFile(path string) (*vdf.Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return vdf.Parse(f)
}

// LoadFile parses a controller config from disk.
func LoadFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Load(f)
}

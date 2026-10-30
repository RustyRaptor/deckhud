package steamcfg

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fakeSteam(t *testing.T) (Locator, string) {
	root := t.TempDir()
	cfg := filepath.Join(root, "steamapps", "common", "Steam Controller Configs", "12345", "config")
	write(t, filepath.Join(cfg, "configset_SERIAL1.vdf"), `"controller_config" { "413080" { "autosave" "1" } }`)
	write(t, filepath.Join(cfg, "configset_controller_neptune.vdf"), `"controller_config" {
		"413080" { "template" "CLOUD_413080/my desktop_0" }
		"620" { "template" "controller_neptune_gamepad_fps.vdf" }
		"1293830" { "autosave" "1" }
		"17480" { "workshop" "607256781" }
	}`)
	write(t, filepath.Join(cfg, "413080", "SERIAL1.vdf"), "autosave")
	write(t, filepath.Join(cfg, "413080", "my desktop_0.vdf"), "cloud")
	write(t, filepath.Join(cfg, "1293830", "controller_neptune.vdf"), "neptune autosave")
	write(t, filepath.Join(root, "controller_base", "templates", "controller_neptune_gamepad_fps.vdf"), "template")
	return Locator{SteamRoot: root}, cfg
}

func TestConfigDirAutodetectsAccount(t *testing.T) {
	l, cfg := fakeSteam(t)
	got, err := l.ConfigDir()
	if err != nil || got != cfg {
		t.Errorf("ConfigDir = %q, %v; want %q", got, err, cfg)
	}
}

func TestResolve(t *testing.T) {
	l, cfg := fakeSteam(t)
	sets := []string{"configset_SERIAL1.vdf", "configset_controller_neptune.vdf"}
	cases := []struct {
		app, want string
	}{
		{"413080", filepath.Join(cfg, "413080", "SERIAL1.vdf")},              // per-controller autosave wins
		{"1293830", filepath.Join(cfg, "1293830", "controller_neptune.vdf")}, // autosave in generic set
		{"620", filepath.Join(l.SteamRoot, "controller_base", "templates", "controller_neptune_gamepad_fps.vdf")},
	}
	for _, c := range cases {
		got, err := l.Resolve(c.app, sets)
		if err != nil || got != c.want {
			t.Errorf("Resolve(%s) = %q, %v; want %q", c.app, got, err, c.want)
		}
	}
	// Without the per-controller set, the cloud template is used.
	got, err := l.Resolve("413080", []string{"configset_controller_neptune.vdf"})
	if want := filepath.Join(cfg, "413080", "my desktop_0.vdf"); err != nil || got != want {
		t.Errorf("Resolve(cloud) = %q, %v; want %q", got, err, want)
	}
	if _, err := l.Resolve("17480", sets); err == nil {
		t.Error("workshop configs are unsupported and should error")
	}
	if _, err := l.Resolve("999", sets); err == nil {
		t.Error("unknown app should error")
	}
}

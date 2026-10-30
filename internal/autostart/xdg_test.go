package autostart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecQuoting(t *testing.T) {
	cases := map[string][]string{
		`/home/deck/Applications/deckhud.AppImage run`: {"/home/deck/Applications/deckhud.AppImage", "run"},
		`"/home/deck/My Apps/deckhud.AppImage" run`:    {"/home/deck/My Apps/deckhud.AppImage", "run"},
		`"/tmp/a\\$b\\"c\\\\d\\` + "`" + `e f" run`:    {"/tmp/a$b\"c\\d`e f", "run"},
	}
	for want, args := range cases {
		if got := Exec(args); got != want {
			t.Errorf("Exec(%q)\n got  %s\n want %s", args, got, want)
		}
	}
}

func TestWriteXDGEnableDisable(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "autostart")
	path, err := WriteXDG(dir, true, []string{"/apps/deckhud.AppImage", "run"})
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(dir, "io.github.RustyRaptor.deckhud.desktop") {
		t.Errorf("path = %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"[Desktop Entry]\n", "Type=Application\n", "Exec=/apps/deckhud.AppImage run\n",
		"Icon=io.github.RustyRaptor.deckhud\n", "OnlyShowIn=KDE;\n", "X-GNOME-Autostart-enabled=true\n",
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("entry lacks %q:\n%s", want, data)
		}
	}

	if _, err := WriteXDG(dir, false, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("disable should remove the entry, stat err = %v", err)
	}
	if _, err := WriteXDG(dir, false, nil); err != nil {
		t.Errorf("disabling twice should be fine: %v", err)
	}
}

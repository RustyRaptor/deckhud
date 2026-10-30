package autostart

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const desktopID = "io.github.RustyRaptor.deckhud.desktop"

// Exec renders args as a Desktop Entry Exec value: arguments with reserved
// characters are double-quoted with ", `, $ and \ escaped, then backslashes
// are escaped again by the general string rule, and % is doubled.
func Exec(args []string) string {
	parts := make([]string, len(args))
	for i, a := range args {
		a = strings.ReplaceAll(a, "%", "%%")
		if strings.ContainsAny(a, " \t\n\"'\\><~|&;$*?#()`") {
			r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`)
			a = `"` + r.Replace(a) + `"`
		}
		parts[i] = strings.ReplaceAll(a, `\`, `\\`)
	}
	return strings.Join(parts, " ")
}

// WriteXDG creates (or with enable=false removes) an XDG autostart entry in
// dir, normally ~/.config/autostart. It returns the entry's path.
func WriteXDG(dir string, enable bool, args []string) (string, error) {
	path := filepath.Join(dir, desktopID)
	if !enable {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return path, err
		}
		return path, nil
	}
	entry := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=deckhud\n" +
		"Comment=On-screen HUD for Steam Input\n" +
		"Exec=" + Exec(args) + "\n" +
		"Icon=io.github.RustyRaptor.deckhud\n" +
		"Terminal=false\n" +
		"OnlyShowIn=KDE;\n" +
		"X-GNOME-Autostart-enabled=true\n"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return path, err
	}
	return path, os.WriteFile(path, []byte(entry), 0o644)
}

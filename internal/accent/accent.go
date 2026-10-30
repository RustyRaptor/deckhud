// Package accent tracks the desktop accent colour.
//
// The source of truth is the XDG settings portal (org.freedesktop.appearance
// accent-color), which works the same on the host and inside a Flatpak and
// signals every change, even when Plasma derives the accent from a wallpaper
// slideshow. kdeglobals is read as a fallback when no portal answers.
package accent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

// FromPortal converts the portal's (r, g, b) doubles in 0..1 to "#rrggbb".
// Out-of-range values mean "no accent set".
func FromPortal(rgb []float64) (string, error) {
	if len(rgb) != 3 {
		return "", fmt.Errorf("accent: want 3 components, got %d", len(rgb))
	}
	var c [3]int
	for i, v := range rgb {
		if v < 0 || v > 1 || math.IsNaN(v) {
			return "", fmt.Errorf("accent: component %v out of range", v)
		}
		c[i] = int(math.Round(v * 255))
	}
	return fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2]), nil
}

// FromKdeglobals reads [General] AccentColor, falling back to the colour
// scheme's selection background.
func FromKdeglobals(r io.Reader) (string, error) {
	var group, accent, selection string
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") {
			group = line
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		switch {
		case !ok:
		case group == "[General]" && k == "AccentColor":
			accent = v
		case group == "[Colors:Selection]" && k == "BackgroundNormal":
			selection = v
		}
	}
	for _, v := range []string{accent, selection} {
		if hex, err := fromRGBList(v); err == nil {
			return hex, nil
		}
	}
	return "", fmt.Errorf("accent: no AccentColor or selection colour in kdeglobals")
}

func fromRGBList(s string) (string, error) {
	parts := strings.Split(s, ",")
	if len(parts) < 3 {
		return "", fmt.Errorf("accent: bad colour %q", s)
	}
	var c [3]int
	for i := range c {
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil || n < 0 || n > 255 {
			return "", fmt.Errorf("accent: bad colour %q", s)
		}
		c[i] = n
	}
	return fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2]), nil
}

const (
	portalDest  = "org.freedesktop.portal.Desktop"
	portalPath  = "/org/freedesktop/portal/desktop"
	settingsIfc = "org.freedesktop.portal.Settings"
	namespace   = "org.freedesktop.appearance"
	key         = "accent-color"
)

// Watch calls f with the current accent and again whenever it changes, until
// ctx ends. It prefers the portal and falls back to polling kdeglobals.
func Watch(ctx context.Context, f func(hex string)) {
	last := ""
	emit := func(hex string) {
		if hex != "" && hex != last {
			last = hex
			f(hex)
		}
	}
	if err := watchPortal(ctx, emit); err == nil || ctx.Err() != nil {
		return
	}
	pollKdeglobals(ctx, emit)
}

func watchPortal(ctx context.Context, emit func(string)) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	obj := conn.Object(portalDest, portalPath)
	var v dbus.Variant
	if err := obj.CallWithContext(ctx, settingsIfc+".ReadOne", 0, namespace, key).Store(&v); err != nil {
		return err
	}
	if hex, err := portalValue(v); err == nil {
		emit(hex)
	}

	if err := conn.AddMatchSignalContext(ctx,
		dbus.WithMatchObjectPath(portalPath),
		dbus.WithMatchInterface(settingsIfc),
		dbus.WithMatchMember("SettingChanged"),
		dbus.WithMatchArg(0, namespace),
		dbus.WithMatchArg(1, key),
	); err != nil {
		return err
	}
	signals := make(chan *dbus.Signal, 16)
	conn.Signal(signals)
	for {
		select {
		case <-ctx.Done():
			return nil
		case s, ok := <-signals:
			if !ok {
				return fmt.Errorf("accent: bus closed")
			}
			if len(s.Body) == 3 {
				if v, ok := s.Body[2].(dbus.Variant); ok {
					if hex, err := portalValue(v); err == nil {
						emit(hex)
					}
				}
			}
		}
	}
}

// portalValue unwraps the (ddd) struct, which godbus decodes as []any.
func portalValue(v dbus.Variant) (string, error) {
	if inner, ok := v.Value().(dbus.Variant); ok { // older portals double-wrap
		v = inner
	}
	fields, ok := v.Value().([]any)
	if !ok {
		return "", fmt.Errorf("accent: unexpected portal value %v", v)
	}
	rgb := make([]float64, 0, 3)
	for _, x := range fields {
		d, ok := x.(float64)
		if !ok {
			return "", fmt.Errorf("accent: unexpected portal value %v", v)
		}
		rgb = append(rgb, d)
	}
	return FromPortal(rgb)
}

func pollKdeglobals(ctx context.Context, emit func(string)) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	path := filepath.Join(dir, "kdeglobals")
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		if f, err := os.Open(path); err == nil {
			if hex, err := FromKdeglobals(f); err == nil {
				emit(hex)
			}
			f.Close()
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

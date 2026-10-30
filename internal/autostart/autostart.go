// Package autostart asks the XDG Background portal to start deckhud with the
// session. Sandboxed apps can't install systemd units or autostart files
// themselves; the portal does it after asking the user.
package autostart

import (
	"context"
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	portalDest = "org.freedesktop.portal.Desktop"
	portalPath = "/org/freedesktop/portal/desktop"
)

// Request enables (or disables) autostart with commandline and reports
// whether the portal granted it.
func Request(ctx context.Context, enable bool, commandline []string) (bool, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	token := "deckhud_autostart"
	sender := conn.Names()[0][1:] // ":1.42" -> "1.42"
	for i := range sender {
		if sender[i] == '.' {
			sender = sender[:i] + "_" + sender[i+1:]
		}
	}
	handle := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + sender + "/" + token)
	if err := conn.AddMatchSignalContext(ctx,
		dbus.WithMatchObjectPath(handle),
		dbus.WithMatchInterface("org.freedesktop.portal.Request"),
		dbus.WithMatchMember("Response"),
	); err != nil {
		return false, err
	}
	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)

	opts := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"reason":       dbus.MakeVariant("Show the Steam Input HUD whenever you use Desktop Mode"),
		"autostart":    dbus.MakeVariant(enable),
		"commandline":  dbus.MakeVariant(commandline),
	}
	var got dbus.ObjectPath
	if err := conn.Object(portalDest, portalPath).CallWithContext(ctx,
		"org.freedesktop.portal.Background.RequestBackground", 0, "", opts).Store(&got); err != nil {
		return false, err
	}
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case s := <-signals:
			if s.Path != got || len(s.Body) < 2 {
				continue
			}
			if code, _ := s.Body[0].(uint32); code != 0 {
				return false, fmt.Errorf("autostart: request denied (response %d)", code)
			}
			results, _ := s.Body[1].(map[string]dbus.Variant)
			granted, _ := results["autostart"].Value().(bool)
			return granted, nil
		}
	}
}

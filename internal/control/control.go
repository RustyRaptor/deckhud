// Package control is a tiny line-based command channel to a running daemon
// over a Unix socket (e.g. "peek" to toggle the cheat sheet). Inside a Flatpak
// the socket lives in the app's shared runtime dir, which every instance of
// the app can reach; loopback TCP is not shared between instances.
package control

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrUnknown is returned by handlers for commands they don't support.
var ErrUnknown = errors.New("unknown command")

// SocketPath returns where the daemon listens.
func SocketPath(runtimeDir, flatpakID string) string {
	if flatpakID != "" {
		return filepath.Join(runtimeDir, "app", flatpakID, "control.sock")
	}
	return filepath.Join(runtimeDir, "deckhud", "control.sock")
}

// DefaultPath derives SocketPath from the environment.
func DefaultPath() string {
	rt := os.Getenv("XDG_RUNTIME_DIR")
	if rt == "" {
		rt = filepath.Join(os.TempDir(), fmt.Sprintf("deckhud-%d", os.Getuid()))
	}
	return SocketPath(rt, os.Getenv("FLATPAK_ID"))
}

// Serve accepts commands until ctx ends. ready is closed once listening.
func Serve(ctx context.Context, path string, ready chan<- struct{}, handle func(cmd string) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if c, err := net.Dial("unix", path); err == nil {
		c.Close()
		return fmt.Errorf("control: another daemon is listening on %s", path)
	}
	os.Remove(path) // stale socket from a crashed daemon
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	go func() { <-ctx.Done(); ln.Close() }()
	close(ready)
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		go func() {
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(2 * time.Second))
			line, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				return
			}
			if err := handle(strings.TrimSpace(line)); err != nil {
				fmt.Fprintf(conn, "error: %v\n", err)
				return
			}
			fmt.Fprintln(conn, "ok")
		}()
	}
}

// Send delivers one command and waits for the reply.
func Send(path, cmd string) error {
	conn, err := net.DialTimeout("unix", path, time.Second)
	if err != nil {
		return fmt.Errorf("control: is deckhudd running? %w", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := fmt.Fprintln(conn, cmd); err != nil {
		return err
	}
	reply, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}
	if reply = strings.TrimSpace(reply); reply != "ok" {
		return errors.New(strings.TrimPrefix(reply, "error: "))
	}
	return nil
}

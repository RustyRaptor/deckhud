package control

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSendReachesHandler(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	got := make(chan string, 1)
	ready := make(chan struct{})
	go func() {
		if err := Serve(ctx, path, ready, func(cmd string) error { got <- cmd; return nil }); err != nil {
			t.Errorf("Serve: %v", err)
		}
	}()
	<-ready
	if err := Send(path, "peek"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	select {
	case cmd := <-got:
		if cmd != "peek" {
			t.Errorf("cmd = %q", cmd)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler not called")
	}
}

func TestUnknownCommandErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan struct{})
	go Serve(ctx, path, ready, func(cmd string) error { return ErrUnknown })
	<-ready
	if err := Send(path, "nope"); err == nil {
		t.Error("expected an error for an unknown command")
	}
}

func TestSendWithoutDaemon(t *testing.T) {
	if err := Send(filepath.Join(t.TempDir(), "none.sock"), "peek"); err == nil {
		t.Error("expected error when no daemon is listening")
	}
}

func TestStaleSocketIsReplaced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "control.sock")
	for i := 0; i < 2; i++ { // second Serve must not fail on the leftover file
		ctx, cancel := context.WithCancel(context.Background())
		ready := make(chan struct{})
		done := make(chan error, 1)
		go func() { done <- Serve(ctx, path, ready, func(string) error { return nil }) }()
		select {
		case <-ready:
		case err := <-done:
			t.Fatalf("round %d: %v", i, err)
		}
		cancel()
		<-done
	}
}

func TestSocketPath(t *testing.T) {
	cases := []struct{ runtime, flatpakID, want string }{
		{"/run/user/1000", "", "/run/user/1000/deckhud/control.sock"},
		{"/run/user/1000", "io.github.RustyRaptor.deckhud", "/run/user/1000/app/io.github.RustyRaptor.deckhud/control.sock"},
	}
	for _, c := range cases {
		if got := SocketPath(c.runtime, c.flatpakID); got != c.want {
			t.Errorf("SocketPath(%q,%q) = %q, want %q", c.runtime, c.flatpakID, got, c.want)
		}
	}
}

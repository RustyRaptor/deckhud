package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/RustyRaptor/deckhud/internal/appwatch"
	"github.com/RustyRaptor/deckhud/internal/engine"
	"github.com/RustyRaptor/deckhud/internal/hid"
	"github.com/RustyRaptor/deckhud/internal/ipc"
	"github.com/RustyRaptor/deckhud/internal/steamcfg"
)

const protocolVersion = 1

type runOpts struct {
	uiDir  string     // directory holding main.qml; empty = don't launch the UI
	print  bool       // print events as JSON lines on stdout
	script []demoStep // play this instead of reading the controller
}

// daemon owns the engine and the config it was built from.
type daemon struct {
	opts runOpts
	hub  *ipc.Hub

	mu      sync.Mutex
	eng     *engine.Engine
	cfg     *steamcfg.Config
	cfgPath string
	cfgMod  time.Time
	app     appwatch.State
}

func run(ctx context.Context, opts runOpts) error {
	d := &daemon{opts: opts}
	token := randomToken()
	d.hub = ipc.NewHub(token, d.greeting)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: d.hub}
	go srv.Serve(ln)
	defer srv.Close()
	url := fmt.Sprintf("ws://%s/%s", ln.Addr(), token)

	if err := d.reload(true); err != nil {
		fmt.Fprintln(os.Stderr, "deckhudd: config:", err)
	}
	go d.watchConfig(ctx)
	ctx, cancel := context.WithCancel(ctx)
	var ui sync.WaitGroup
	defer func() { cancel(); ui.Wait() }() // never leave an orphaned overlay
	if opts.uiDir != "" {
		ui.Go(func() { superviseUI(ctx, opts.uiDir, url) })
	}
	if opts.script != nil {
		d.playScript(ctx, opts.script)
		return nil
	}
	d.readController(ctx)
	return nil
}

func randomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func (d *daemon) greeting() []any {
	d.mu.Lock()
	defer d.mu.Unlock()
	msgs := []any{map[string]any{"type": "hello", "version": protocolVersion}}
	if d.eng != nil {
		msgs = append(msgs, d.configMsg())
		st := d.eng.State()
		msgs = append(msgs, engine.Event{Type: "state", State: &st})
	}
	return msgs
}

func (d *daemon) configMsg() map[string]any {
	return map[string]any{"type": "config", "app": d.app.App, "title": d.cfg.Title}
}

func (d *daemon) emit(v any) {
	d.hub.Broadcast(v)
	if d.opts.print {
		if data, err := json.Marshal(v); err == nil {
			fmt.Println(string(data))
		}
	}
}

// reload re-reads Steam's log and the config it points at. With full=false it
// only follows log lines appended since the last call.
func (d *daemon) reload(full bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	changed := false
	if full {
		d.app = appwatch.State{}
		data, err := os.ReadFile(logPath())
		if err != nil {
			return err
		}
		for _, line := range splitLines(string(data)) {
			d.app.Feed(line)
		}
		d.app.Flush()
		changed = true
	}
	path, err := steamcfg.Locator{SteamRoot: steamRoot()}.Resolve(d.app.App, d.app.ConfigSets)
	if err != nil {
		return err
	}
	st, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !changed && path == d.cfgPath && st.ModTime().Equal(d.cfgMod) {
		return nil
	}
	cfg, err := steamcfg.LoadFile(path)
	if err != nil {
		return err
	}
	d.cfg, d.cfgPath, d.cfgMod = cfg, path, st.ModTime()
	d.eng = engine.New(cfg, engine.Options{})
	st2 := d.eng.State()
	go func(cfgMsg map[string]any, state engine.Event) { // emit outside the lock
		d.emit(cfgMsg)
		d.emit(state)
	}(d.configMsg(), engine.Event{Type: "state", State: &st2})
	return nil
}

func (d *daemon) watchConfig(ctx context.Context) {
	tail := &appwatch.Tail{Path: logPath()}
	tail.Poll() // skip history; reload(true) already replayed it
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		lines, _ := tail.Poll()
		d.mu.Lock()
		for _, l := range lines {
			d.app.Feed(l)
		}
		d.app.Flush()
		d.mu.Unlock()
		if err := d.reload(false); err != nil {
			fmt.Fprintln(os.Stderr, "deckhudd: config:", err)
		}
	}
}

// readController streams controller states into the engine, reopening the
// device after errors (e.g. suspend/resume).
func (d *daemon) readController(ctx context.Context) {
	for ctx.Err() == nil {
		f, err := openDevice(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, "deckhudd: controller:", err)
		} else {
			err = hid.Stream(f, d.feed)
			f.Close()
			if err != nil && ctx.Err() == nil {
				fmt.Fprintln(os.Stderr, "deckhudd: controller:", err)
			}
		}
		select {
		case <-ctx.Done():
		case <-time.After(2 * time.Second):
		}
	}
}

// superviseUI keeps the QML overlay running, restarting it if it exits.
func superviseUI(ctx context.Context, dir, url string) {
	qml := "/usr/lib/qt6/bin/qml"
	for ctx.Err() == nil {
		cmd := exec.CommandContext(ctx, qml, filepath.Join(dir, "main.qml"), "--", url)
		cmd.Env = append(os.Environ(), "QT_FORCE_STDERR_LOGGING=1")
		cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
		cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
		cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
		cmd.WaitDelay = 2 * time.Second
		if err := cmd.Run(); err != nil && ctx.Err() == nil {
			fmt.Fprintln(os.Stderr, "deckhudd: ui exited:", err)
		}
		select {
		case <-ctx.Done():
		case <-time.After(2 * time.Second):
		}
	}
}

func logPath() string { return filepath.Join(steamRoot(), "logs", "controller.txt") }

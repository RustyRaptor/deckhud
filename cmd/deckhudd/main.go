// Command deckhudd is the deckhud daemon. During early development it also
// carries diagnostic subcommands.
package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/RustyRaptor/deckhud/internal/appwatch"
	"github.com/RustyRaptor/deckhud/internal/autostart"
	"github.com/RustyRaptor/deckhud/internal/binding"
	"github.com/RustyRaptor/deckhud/internal/control"
	"github.com/RustyRaptor/deckhud/internal/hid"
	"github.com/RustyRaptor/deckhud/internal/steamcfg"
)

const usage = `usage: deckhudd <command>

commands:
  run [ui-dir]    (default) run the daemon and launch the overlay from ui-dir
                  (default: ../share/deckhud/ui next to the binary)
  events          run without the overlay, printing events as JSON lines
  demo [ui-dir]   play a scripted session through the overlay (no controller needed)
  autostart on|off  start with Plasma via the Background portal (Flatpak);
                  native installs use the systemd user unit instead
  peek            toggle the cheat sheet in the running daemon
                  (bind it to a KDE shortcut, then map a Steam button to that)
  find            print the Deck controller hidraw device
  dump            print the active Steam Input config as deckhud sees it
  monitor         print button changes and stick angles live
  record <file>   save raw controller reports to <file> (Ctrl+C to stop)
`

func main() {
	if len(os.Args) < 2 {
		os.Args = append(os.Args, "run") // launched from the menu or autostart
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var err error
	switch os.Args[1] {
	case "run":
		dir := defaultUIDir()
		if len(os.Args) > 2 {
			dir = os.Args[2]
		}
		err = run(ctx, runOpts{uiDir: dir, autostart: true})
	case "demo":
		dir := defaultUIDir()
		if len(os.Args) > 2 {
			dir = os.Args[2]
		}
		err = demo(ctx, dir)
	case "events":
		err = run(ctx, runOpts{print: true})
	case "autostart":
		enable := len(os.Args) < 3 || os.Args[2] != "off"
		var granted bool
		if granted, err = autostart.Request(ctx, enable, []string{"deckhudd", "run"}); err == nil {
			fmt.Println("autostart:", map[bool]string{true: "on", false: "off"}[granted])
		}
	case "peek":
		err = control.Send(control.DefaultPath(), "peek")
	case "find":
		var dev string
		if dev, err = hid.Find("/sys"); err == nil {
			fmt.Println(dev)
		}
	case "dump":
		err = dump()
	case "monitor":
		err = monitor(ctx)
	case "record":
		if len(os.Args) < 3 {
			fmt.Fprint(os.Stderr, usage)
			os.Exit(2)
		}
		err = record(ctx, os.Args[2])
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "deckhudd:", err)
		os.Exit(1)
	}
}

// openDevice opens the controller and closes it when ctx ends, which unblocks
// any pending read.
func openDevice(ctx context.Context) (*os.File, error) {
	dev, err := hid.Find("/sys")
	if err != nil {
		return nil, err
	}
	f, err := os.Open(dev)
	if err != nil {
		return nil, err
	}
	go func() { <-ctx.Done(); f.Close() }()
	return f, nil
}

func monitor(ctx context.Context) error {
	f, err := openDevice(ctx)
	if err != nil {
		return err
	}
	fmt.Println("press buttons / move sticks; Ctrl+C to quit")
	var prev hid.State
	var lastStick time.Time
	err = hid.Stream(f, func(s hid.State) {
		if pressed, released := prev.Buttons.Diff(s.Buttons); pressed|released != 0 {
			pressed.Each(func(b hid.Button) {
				in, _ := b.SteamInput()
				fmt.Printf("down %-12s steam=%s/%s\n", b, in.Source, in.Name)
			})
			released.Each(func(b hid.Button) { fmt.Printf("up   %s\n", b) })
		}
		if time.Since(lastStick) > 150*time.Millisecond {
			for _, st := range []struct {
				name string
				v    hid.Vec
			}{{"left", s.LeftStick}, {"right", s.RightStick}} {
				if mag := math.Hypot(float64(st.v.X), float64(st.v.Y)); mag > 8000 {
					deg := math.Atan2(float64(st.v.X), float64(st.v.Y)) * 180 / math.Pi
					fmt.Printf("%s stick: %4.0f° (0=up, clockwise) mag %5.0f\n", st.name, math.Mod(deg+360, 360), mag)
					lastStick = time.Now()
				}
			}
		}
		prev = s
	})
	if ctx.Err() != nil {
		return nil
	}
	return err
}

func record(ctx context.Context, path string) error {
	f, err := openDevice(ctx)
	if err != nil {
		return err
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	n, err := io.Copy(out, f)
	fmt.Fprintf(os.Stderr, "recorded %d reports\n", n/hid.ReportSize)
	if ctx.Err() != nil {
		return nil
	}
	return err
}

// steamRoot finds the Steam data directory of a native or Flatpak Steam.
func steamRoot() string {
	home, _ := os.UserHomeDir()
	native := filepath.Join(home, ".local", "share", "Steam")
	flatpak := filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam")
	if _, err := os.Stat(native); err != nil {
		if _, err := os.Stat(flatpak); err == nil {
			return flatpak
		}
	}
	return native
}

// activeConfig replays Steam's controller log and loads the config it points at.
func activeConfig() (*steamcfg.Config, string, appwatch.State, error) {
	var st appwatch.State
	log, err := os.ReadFile(filepath.Join(steamRoot(), "logs", "controller.txt"))
	if err != nil {
		return nil, "", st, err
	}
	for _, line := range splitLines(string(log)) {
		st.Feed(line)
	}
	st.Flush()
	path, err := steamcfg.Locator{SteamRoot: steamRoot()}.Resolve(st.App, st.ConfigSets)
	if err != nil {
		return nil, "", st, err
	}
	cfg, err := steamcfg.LoadFile(path)
	return cfg, path, st, err
}

func dump() error {
	cfg, path, st, err := activeConfig()
	if err != nil {
		return err
	}
	fmt.Printf("app %s via %v\nconfig %q (%s)\n", st.App, st.ConfigSets, cfg.Title, filepath.Base(path))
	for _, set := range cfg.Sets {
		kind := "action set"
		if set.Layer {
			kind = "layer of " + set.Parent
		}
		fmt.Printf("\n[%d] %s (%s)\n", set.ID+1, set.Title, kind)
		var layers []*steamcfg.Set
		base := set
		if set.Layer {
			layers, base = []*steamcfg.Set{set}, cfg.Base()
		}
		sources := map[string]bool{}
		for _, sb := range set.Sources {
			sources[sb.Source] = true
		}
		var names []string
		for n := range sources {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, src := range names {
			g := cfg.ActiveGroup(base, layers, false, src)
			if g == nil {
				continue
			}
			fmt.Printf("  %-15s %s\n", src, g.Mode)
			var inputs []string
			for in := range g.Inputs {
				inputs = append(inputs, in)
			}
			sort.Strings(inputs)
			for _, in := range inputs {
				for _, a := range cfg.Activators(base, layers, false, src, in) {
					fmt.Printf("      %-26s %-12s %s\n", in, a.Kind, strings.Join(binding.Chord(a.Bindings), " + "))
				}
			}
		}
	}
	return nil
}

func splitLines(s string) []string { return strings.Split(s, "\n") }

func defaultUIDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "ui"
	}
	return filepath.Join(filepath.Dir(exe), "..", "share", "deckhud", "ui")
}

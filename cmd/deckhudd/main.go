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
	"time"

	"github.com/RustyRaptor/deckhud/internal/hid"
)

const usage = `usage: deckhudd <command>

commands:
  find            print the Deck controller hidraw device
  monitor         print button changes and stick angles live
  record <file>   save raw controller reports to <file> (Ctrl+C to stop)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var err error
	switch os.Args[1] {
	case "find":
		var dev string
		if dev, err = hid.Find("/sys"); err == nil {
			fmt.Println(dev)
		}
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

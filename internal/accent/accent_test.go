package accent

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFromPortal(t *testing.T) {
	cases := []struct {
		in      []float64
		want    string
		wantErr bool
	}{
		{[]float64{0.941176, 0.611765, 0.792157}, "#f09cca", false},
		{[]float64{0, 0, 0}, "#000000", false},
		{[]float64{1, 1, 1}, "#ffffff", false},
		{[]float64{-1, -1, -1}, "", true}, // portal's "no accent set"
		{[]float64{0.5, 2, 0.5}, "", true},
		{[]float64{0.5, 0.5}, "", true},
	}
	for _, c := range cases {
		got, err := FromPortal(c.in)
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("FromPortal(%v) = %q, %v; want %q, err=%v", c.in, got, err, c.want, c.wantErr)
		}
	}
}

func TestFromKdeglobals(t *testing.T) {
	src := `[ColorEffects:Disabled]
AccentColor=1,2,3

[Colors:Selection]
BackgroundNormal=126,196,124

[General]
AccentColor=243,121,185
accentColorFromWallpaper=true
`
	got, err := FromKdeglobals(strings.NewReader(src))
	if err != nil || got != "#f379b9" {
		t.Errorf("got %q, %v; want #f379b9 from [General] only", got, err)
	}

	noAccent := "[Colors:Selection]\nBackgroundNormal=61,174,233\n"
	if got, err := FromKdeglobals(strings.NewReader(noAccent)); err != nil || got != "#3daee9" {
		t.Errorf("selection fallback = %q, %v", got, err)
	}
	if _, err := FromKdeglobals(strings.NewReader("[General]\nfoo=bar\n")); err == nil {
		t.Error("expected error with no colour at all")
	}
}

// TestWatchLivePortal talks to the real session portal; skipped without one.
func TestWatchLivePortal(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("no session bus")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	got := make(chan string, 8)
	go Watch(ctx, func(hex string) { got <- hex })
	select {
	case c := <-got:
		if len(c) != 7 || c[0] != '#' {
			t.Errorf("bad colour %q", c)
		}
	case <-ctx.Done():
		t.Skip("portal did not answer (no xdg-desktop-portal?)")
	}
}

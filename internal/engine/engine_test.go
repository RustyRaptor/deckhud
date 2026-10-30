package engine

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/RustyRaptor/deckhud/internal/hid"
	"github.com/RustyRaptor/deckhud/internal/steamcfg"
)

const tick = 4 * time.Millisecond // the Deck reports at ~250 Hz

// sim drives an Engine like the real device would.
type sim struct {
	t      *testing.T
	e      *Engine
	now    time.Time
	state  hid.State
	events []Event
}

func newSim(t *testing.T, fixture string) *sim {
	t.Helper()
	f, err := os.Open("../../testdata/vdf/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := steamcfg.Load(f)
	if err != nil {
		t.Fatal(err)
	}
	return &sim{t: t, e: New(cfg, Options{}), now: time.Unix(1000, 0)}
}

func (s *sim) step() {
	s.now = s.now.Add(tick)
	s.state.Seq++
	s.events = append(s.events, s.e.Update(s.state, s.now)...)
}

func (s *sim) press(b hid.Button)   { s.state.Buttons |= b; s.step() }
func (s *sim) release(b hid.Button) { s.state.Buttons &^= b; s.step() }
func (s *sim) rstick(x, y int16)    { s.state.RightStick = hid.Vec{X: x, Y: y}; s.step() }
func (s *sim) wait(d time.Duration) {
	for end := s.now.Add(d); s.now.Before(end); {
		s.step()
	}
}

// take returns and clears the events collected so far.
func (s *sim) take() []Event {
	ev := s.events
	s.events = nil
	return ev
}

func chords(evs []Event) []Chord {
	var out []Chord
	for _, e := range evs {
		if e.Chord != nil {
			out = append(out, *e.Chord)
		}
	}
	return out
}

func states(evs []Event) []SetState {
	var out []SetState
	for _, e := range evs {
		if e.State != nil {
			out = append(out, *e.State)
		}
	}
	return out
}

func radials(evs []Event) []Radial {
	var out []Radial
	for _, e := range evs {
		if e.Radial != nil {
			out = append(out, *e.Radial)
		}
	}
	return out
}

func TestFullPressFiresOnPress(t *testing.T) {
	s := newSim(t, "layers.vdf")
	s.press(hid.L5)
	got := chords(s.take())
	want := []Chord{{Button: "L5", Input: "button_back_left", Activator: "Full_Press", Keys: []string{"Space"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
	s.release(hid.L5)
	if ev := s.take(); len(ev) != 0 {
		t.Errorf("release should be silent, got %+v", ev)
	}
}

func TestEventTypeIsSet(t *testing.T) {
	s := newSim(t, "layers.vdf")
	s.press(hid.L5)
	if ev := s.take(); len(ev) != 1 || ev[0].Type != "chord" {
		t.Errorf("events = %+v", ev)
	}
}

func TestShortPressWithLongPressFiresFullOnRelease(t *testing.T) {
	s := newSim(t, "layers.vdf")
	s.press(hid.R5)
	s.wait(100 * time.Millisecond)
	if ev := s.take(); len(ev) != 0 {
		t.Fatalf("nothing should fire while a long press is still possible: %+v", ev)
	}
	s.release(hid.R5)
	got := chords(s.take())
	if len(got) != 1 || got[0].Activator != "Full_Press" || !reflect.DeepEqual(got[0].Keys, []string{"Ctrl", "V"}) {
		t.Errorf("got %+v", got)
	}
}

func TestLongPressHoldLayer(t *testing.T) {
	s := newSim(t, "layers.vdf")
	s.press(hid.R5)
	s.wait(450 * time.Millisecond) // fixture long_press_time is 400ms
	ev := s.take()
	if c := chords(ev); len(c) != 0 {
		t.Errorf("hold_layer should not show a chord: %+v", c)
	}
	if st := states(ev); !reflect.DeepEqual(st, []SetState{{Set: "Desktop", Layers: []string{"Editing"}}}) {
		t.Fatalf("states = %+v", st)
	}

	// The layer remaps L5 to Ctrl+Z, labelled "Undo".
	s.press(hid.L5)
	got := chords(s.take())
	if len(got) != 1 || got[0].Label != "Undo" || !reflect.DeepEqual(got[0].Keys, []string{"Ctrl", "Z"}) {
		t.Errorf("layered L5 = %+v", got)
	}
	s.release(hid.L5)

	s.release(hid.R5)
	ev = s.take()
	if st := states(ev); !reflect.DeepEqual(st, []SetState{{Set: "Desktop", Layers: nil}}) {
		t.Errorf("releasing the hold should drop the layer, got %+v", st)
	}
	if c := chords(ev); len(c) != 0 {
		t.Errorf("Full_Press must not fire after a long press: %+v", c)
	}
}

func TestModeShiftRadialMenu(t *testing.T) {
	s := newSim(t, "layers.vdf")
	// Without the mode shift the right stick is a mouse: no radial.
	s.rstick(0, 30000)
	s.rstick(0, 0)
	if r := radials(s.take()); len(r) != 0 {
		t.Fatalf("unexpected radial: %+v", r)
	}

	s.press(hid.L1) // mode_shift right_joystick -> radial menu
	if c := chords(s.take()); len(c) != 0 {
		t.Errorf("mode shift should not show a chord: %+v", c)
	}
	items := []MenuItem{
		{Label: "Copy", Keys: []string{"Ctrl", "C"}},
		{Label: "Paste", Icon: "ghost_045_move_0010.png", Keys: []string{"Ctrl", "V"}},
	}

	s.rstick(0, 30000) // up: item 1 of 2
	got := radials(s.take())
	want := []Radial{{Phase: "open", Source: "right_joystick", Title: "text ops", Items: items, Index: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("open:\n got  %+v\n want %+v", got, want)
	}

	s.rstick(0, 30000) // no movement, no event
	s.rstick(0, -30000)
	got = radials(s.take())
	if len(got) != 1 || got[0].Phase != "move" || got[0].Index != 1 {
		t.Fatalf("move: %+v", got)
	}

	s.rstick(0, 0)
	got = radials(s.take())
	if len(got) != 1 || got[0].Phase != "fire" || got[0].Index != 1 || !reflect.DeepEqual(got[0].Keys, []string{"Ctrl", "V"}) {
		t.Fatalf("fire: %+v", got)
	}
	s.release(hid.L1)
	if ev := s.take(); len(ev) != 0 {
		t.Errorf("closed menu should stay quiet: %+v", ev)
	}
}

func TestRadialCancelledWhenModeShiftReleased(t *testing.T) {
	s := newSim(t, "layers.vdf")
	s.press(hid.L1)
	s.rstick(0, 30000)
	s.take()
	s.release(hid.L1)
	got := radials(s.take())
	if len(got) != 1 || got[0].Phase != "cancel" {
		t.Errorf("got %+v", got)
	}
}

func TestRealDesktopRadialWedges(t *testing.T) {
	// 7 items, item 1 centred at the top, clockwise.
	cases := []struct {
		x, y int16
		want int
	}{
		{0, 30000, 0},
		{30000, 0, 2},     // 90° -> wedge 2 (77.1°..128.6°)
		{5209, -29544, 3}, // 170° -> wedge 3 (128.6°..180°)
		{-30000, 0, 5},    // 270°
		{-8000, 30000, 0}, // ~345°, wraps to item 1
	}
	for _, c := range cases {
		s := newSim(t, "desktop_neptune.vdf")
		s.rstick(c.x, c.y)
		got := radials(s.take())
		if len(got) != 1 || got[0].Phase != "open" || got[0].Index != c.want || len(got[0].Items) != 7 {
			t.Errorf("stick (%d,%d): %+v", c.x, c.y, got)
		}
	}
}

func TestSmallDeflectionDoesNotOpen(t *testing.T) {
	s := newSim(t, "desktop_neptune.vdf")
	s.rstick(3000, -2000) // stick noise
	if ev := s.take(); len(ev) != 0 {
		t.Errorf("got %+v", ev)
	}
}

func TestDoublePress(t *testing.T) {
	s := newSim(t, "desktop_neptune.vdf")
	// button_back_right (L5/R5 lower): Full = Ctrl+V, Double = Ctrl+Shift+V
	s.press(hid.R5)
	s.release(hid.R5)
	s.wait(50 * time.Millisecond)
	s.press(hid.R5)
	s.release(hid.R5)
	s.wait(400 * time.Millisecond)
	got := chords(s.take())
	if len(got) != 1 || got[0].Activator != "Double_Press" || !reflect.DeepEqual(got[0].Keys, []string{"Ctrl", "Shift", "V"}) {
		t.Errorf("double: %+v", got)
	}

	s.press(hid.R5)
	s.release(hid.R5)
	if c := chords(s.take()); len(c) != 0 {
		t.Fatalf("single press must wait out the double-press window: %+v", c)
	}
	s.wait(400 * time.Millisecond)
	got = chords(s.take())
	if len(got) != 1 || got[0].Activator != "Full_Press" || !reflect.DeepEqual(got[0].Keys, []string{"Ctrl", "V"}) {
		t.Errorf("single: %+v", got)
	}
}

func TestChangePresetSwitchesActionSet(t *testing.T) {
	s := newSim(t, "desktop_neptune.vdf")
	s.press(hid.L4) // button_back_left_upper: Long_Press -> CHANGE_PRESET 2
	s.wait(600 * time.Millisecond)
	if st := states(s.take()); !reflect.DeepEqual(st, []SetState{{Set: "Gamepad"}}) {
		t.Fatalf("states = %+v", st)
	}
	s.release(hid.L4)
	s.take()
	s.press(hid.A)
	got := chords(s.take())
	if len(got) != 1 || !reflect.DeepEqual(got[0].Keys, []string{"Pad A"}) {
		t.Errorf("A in Gamepad set = %+v", got)
	}
	if s.e.State().Set != "Gamepad" {
		t.Errorf("State() = %+v", s.e.State())
	}
}

func TestUnmappedButtonsAreSilent(t *testing.T) {
	s := newSim(t, "layers.vdf")
	s.press(hid.Steam | hid.QAM | hid.B)
	s.release(hid.Steam | hid.QAM | hid.B)
	if ev := s.take(); len(ev) != 0 {
		t.Errorf("got %+v", ev)
	}
}

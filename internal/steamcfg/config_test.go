package steamcfg

import (
	"os"
	"testing"

	"github.com/RustyRaptor/deckhud/internal/binding"
)

func load(t *testing.T, name string) *Config {
	t.Helper()
	f, err := os.Open("../../testdata/vdf/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	c, err := Load(f)
	if err != nil {
		t.Fatalf("Load(%s): %v", name, err)
	}
	return c
}

func TestLoadSetsAndLayers(t *testing.T) {
	c := load(t, "layers.vdf")
	if c.Title != "layers fixture" {
		t.Errorf("Title = %q", c.Title)
	}
	if len(c.Sets) != 3 {
		t.Fatalf("got %d sets, want 3", len(c.Sets))
	}
	want := []Set{
		{ID: 0, Key: "Default", Title: "Desktop"},
		{ID: 1, Key: "Preset_1000001", Title: "Gamepad"},
		{ID: 2, Key: "Preset_1000002", Title: "Editing", Layer: true, Parent: "Default"},
	}
	for i, w := range want {
		g := c.Sets[i]
		if g.ID != w.ID || g.Key != w.Key || g.Title != w.Title || g.Layer != w.Layer || g.Parent != w.Parent {
			t.Errorf("set %d = %+v, want %+v", i, *g, w)
		}
	}
	if c.Base().Key != "Default" {
		t.Errorf("Base = %q", c.Base().Key)
	}
	// Binding targets are 1-based preset ids: "hold_layer 3" means preset id 2.
	if s := c.SetByTarget(3); s == nil || s.Key != "Preset_1000002" {
		t.Errorf("SetByTarget(3) = %+v", s)
	}
	if c.SetByTarget(9) != nil || c.SetByTarget(0) != nil {
		t.Error("out-of-range targets should be nil")
	}
}

func TestSourceBindings(t *testing.T) {
	c := load(t, "layers.vdf")
	got := c.SetByTarget(1).Sources
	want := []SourceBinding{
		{GroupID: 0, Source: "switch", Active: true},
		{GroupID: 1, Source: "right_joystick", Active: true},
		{GroupID: 3, Source: "right_joystick", Active: true, ModeShift: true},
		{GroupID: 5, Source: "button_diamond", Active: false},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d sources, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("source %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestGroupsAndActivators(t *testing.T) {
	c := load(t, "layers.vdf")
	g := c.Groups[0]
	if g == nil || g.Mode != "switches" || g.Layer {
		t.Fatalf("group 0 = %+v", g)
	}
	acts := g.Inputs["button_back_right"]
	if len(acts) != 2 {
		t.Fatalf("button_back_right has %d activators, want 2", len(acts))
	}
	if acts[0].Kind != "Full_Press" || len(acts[0].Bindings) != 2 || acts[0].Bindings[1].Args[0] != "V" {
		t.Errorf("Full_Press = %+v", acts[0])
	}
	if acts[1].Kind != "Long_Press" || acts[1].Settings["long_press_time"] != "400" {
		t.Errorf("Long_Press = %+v", acts[1])
	}
	if sc, ok := acts[1].Bindings[0].StateChange(); !ok || sc.Op != binding.HoldLayer || sc.Target != 3 {
		t.Errorf("Long_Press state change = %+v %v", sc, ok)
	}
	if !c.Groups[2].Layer {
		t.Error("group 2 should be a layer group")
	}
}

func TestMenuItemsOrderedByIndex(t *testing.T) {
	c := load(t, "layers.vdf")
	items := c.Groups[3].MenuItems()
	if len(items) != 2 {
		t.Fatalf("got %d items", len(items))
	}
	if items[0].Index != 1 || items[0].Label != "Copy" || items[1].Index != 2 || items[1].Label != "Paste" {
		t.Errorf("items = %+v", items)
	}
	if items[1].Icon != "ghost_045_move_0010.png" {
		t.Errorf("icon = %q", items[1].Icon)
	}
	if got := binding.Chord(items[0].Bindings); len(got) != 2 || got[0] != "Ctrl" {
		t.Errorf("chord = %v", got)
	}
}

func TestResolveWithLayersAndModeShift(t *testing.T) {
	c := load(t, "layers.vdf")
	base := c.Base()
	layer := c.SetByTarget(3)

	// Base set: R4 is Ctrl+V; L5 is Space.
	if acts := c.Activators(base, nil, false, "switch", "button_back_right"); len(acts) != 2 {
		t.Errorf("base R4 activators = %+v", acts)
	}
	if acts := c.Activators(base, nil, false, "switch", "button_back_left"); acts[0].Bindings[0].Args[0] != "SPACE" {
		t.Errorf("base L5 = %+v", acts)
	}

	// With the layer: L5 overridden, R4 falls through to the base group.
	stack := []*Set{layer}
	if acts := c.Activators(base, stack, false, "switch", "button_back_left"); acts[0].Bindings[1].Args[0] != "Z" {
		t.Errorf("layer L5 = %+v", acts)
	}
	if acts := c.Activators(base, stack, false, "switch", "button_back_right"); len(acts) != 2 {
		t.Errorf("layer R4 should fall through, got %+v", acts)
	}

	// Inactive groups are ignored; unknown inputs are nil.
	if acts := c.Activators(base, nil, false, "button_diamond", "button_a"); acts != nil {
		t.Errorf("inactive group used: %+v", acts)
	}
	if acts := c.Activators(base, nil, false, "switch", "nope"); acts != nil {
		t.Errorf("unknown input: %+v", acts)
	}

	// Group selection: mode shift swaps right_joystick to the radial menu.
	if g := c.ActiveGroup(base, nil, false, "right_joystick"); g == nil || g.ID != 1 {
		t.Errorf("right stick group = %+v", g)
	}
	if g := c.ActiveGroup(base, nil, true, "right_joystick"); g == nil || g.ID != 3 {
		t.Errorf("mode-shifted right stick group = %+v", g)
	}
	// An empty layer group of a *different* mode replaces the source outright.
	if g := c.ActiveGroup(base, stack, false, "right_joystick"); g == nil || g.ID != 4 {
		t.Errorf("layered right stick group = %+v", g)
	}
}

func TestRealDesktopConfig(t *testing.T) {
	c := load(t, "desktop_neptune.vdf")
	if c.Title != "d4sktop dogeeeee" || c.Base().Title != "Desktop" {
		t.Fatalf("title %q base %q", c.Title, c.Base().Title)
	}
	g := c.ActiveGroup(c.Base(), nil, false, "right_joystick")
	if g == nil || g.Mode != "radial_menu" {
		t.Fatalf("right stick should be a radial menu, got %+v", g)
	}
	if len(g.MenuItems()) == 0 {
		t.Error("radial menu has no items")
	}
	acts := c.Activators(c.Base(), nil, false, "switch", "button_back_right_upper")
	if len(acts) == 0 {
		t.Fatal("R4 has no activators")
	}
	if got := binding.Chord(acts[0].Bindings); len(got) != 3 || got[0] != "Meta" || got[1] != "Shift" || got[2] != "." {
		t.Errorf("R4 chord = %v, want Meta Shift .", got)
	}
	// "CHANGE_PRESET 2" switches to the Gamepad set.
	if s := c.SetByTarget(2); s == nil || s.Title != "Gamepad" {
		t.Errorf("SetByTarget(2) = %+v", s)
	}
}

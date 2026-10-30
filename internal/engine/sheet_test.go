package engine

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/RustyRaptor/deckhud/internal/hid"
)

func control(sh Sheet, glyph string) *Control {
	for i := range sh.Controls {
		if sh.Controls[i].Glyph == glyph {
			return &sh.Controls[i]
		}
	}
	return nil
}

func TestSheetBaseSet(t *testing.T) {
	s := newSim(t, "desktop_neptune.vdf")
	sh := s.e.Sheet()
	if sh.Set != "Desktop" || len(sh.Layers) != 0 {
		t.Fatalf("set/layers = %q %v", sh.Set, sh.Layers)
	}

	r4 := control(sh, "R4")
	if r4 == nil {
		t.Fatal("R4 missing")
	}
	want := []Entry{{Activator: "Full_Press", Keys: []string{"Meta", "Shift", "."}}}
	if !reflect.DeepEqual(r4.Entries, want) || r4.Changed {
		t.Errorf("R4 = %+v", *r4)
	}

	view := control(sh, "View")
	if view == nil || len(view.Entries) != 3 {
		t.Fatalf("View = %+v", view)
	}

	rs := control(sh, "RS")
	if rs == nil || rs.Mode != "radial_menu" || len(rs.Entries) != 7 {
		t.Fatalf("right stick = %+v", rs)
	}
	if rs.Entries[0].Input != "touch_menu_button_1" || !reflect.DeepEqual(rs.Entries[0].Keys, []string{"Ctrl", "V"}) {
		t.Errorf("first radial entry = %+v", rs.Entries[0])
	}

	// Unbound physical controls are left out; Steam/QAM are never listed.
	for _, g := range []string{"Steam", "QAM"} {
		if control(sh, g) != nil {
			t.Errorf("%s should not be listed", g)
		}
	}
}

func TestSheetMarksLayerChanges(t *testing.T) {
	s := newSim(t, "layers.vdf")
	s.press(hid.R5)
	s.wait(450 * time.Millisecond) // hold_layer "Editing"
	sh := s.e.Sheet()
	if !reflect.DeepEqual(sh.Layers, []string{"Editing"}) {
		t.Fatalf("layers = %v", sh.Layers)
	}
	l5 := control(sh, "L5")
	if l5 == nil || !l5.Changed || l5.Entries[0].Label != "Undo" {
		t.Errorf("L5 = %+v", l5)
	}
	if r5 := control(sh, "R5"); r5 == nil || r5.Changed {
		t.Errorf("R5 falls through and is unchanged: %+v", r5)
	}
	if rs := control(sh, "RS"); rs == nil || !rs.Changed || rs.Mode != "radial_menu" {
		t.Errorf("RS mode changed by the layer: %+v", rs)
	}
}

func TestSheetOrderIsStable(t *testing.T) {
	s := newSim(t, "desktop_neptune.vdf")
	a, b := s.e.Sheet(), s.e.Sheet()
	if !reflect.DeepEqual(a, b) {
		t.Error("Sheet must be deterministic")
	}
	var glyphs []string
	for _, c := range a.Controls {
		glyphs = append(glyphs, c.Glyph)
	}
	// Left side first, then face buttons, then the right side.
	idx := func(g string) int {
		for i, x := range glyphs {
			if x == g {
				return i
			}
		}
		return -1
	}
	if !(idx("L1") < idx("A") && idx("A") < idx("R1")) {
		t.Errorf("order = %v", glyphs)
	}
}

func TestStateEventsCarryTheSheet(t *testing.T) {
	s := newSim(t, "layers.vdf")
	s.press(hid.R5)
	s.wait(450 * time.Millisecond)
	for _, ev := range s.take() {
		if ev.Type == "state" {
			if ev.Sheet == nil || !reflect.DeepEqual(ev.Sheet.Layers, []string{"Editing"}) {
				t.Errorf("state event sheet = %+v", ev.Sheet)
			}
			return
		}
	}
	t.Fatal("no state event")
}

func TestSheetJSONHasNoNullEntries(t *testing.T) {
	s := newSim(t, "desktop_neptune.vdf")
	data, err := json.Marshal(s.e.Sheet())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"entries":null`) {
		t.Errorf("controls without inputs must encode entries as [], got %s", data)
	}
}

package engine

import (
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/RustyRaptor/deckhud/internal/binding"
	"github.com/RustyRaptor/deckhud/internal/steamcfg"
)

// Sheet is a snapshot of what every control does right now, for the
// layer cheat sheet.
type Sheet struct {
	Set      string    `json:"set"`
	Layers   []string  `json:"layers,omitempty"`
	Controls []Control `json:"controls"`
}

// Control is one physical control (or a whole stick/pad/d-pad) and its bindings.
type Control struct {
	Glyph   string  `json:"glyph"`
	Source  string  `json:"source"`
	Mode    string  `json:"mode"`
	Changed bool    `json:"changed,omitempty"` // differs from the bare action set
	Entries []Entry `json:"entries"`
}

// Entry is one activator of one input.
type Entry struct {
	Input     string   `json:"input,omitempty"` // set for multi-input controls
	Activator string   `json:"activator"`
	Label     string   `json:"label,omitempty"`
	Keys      []string `json:"keys"`
}

// layout lists controls roughly as they sit on the Deck, left to right. An
// empty input means the control is the whole group bound to source.
var layout = []struct{ glyph, source, input string }{
	{"L2", "left_trigger", ""},
	{"L1", "switch", "left_bumper"},
	{"L4", "switch", "button_back_left_upper"},
	{"L5", "switch", "button_back_left"},
	{"View", "switch", "button_escape"},
	{"LS", "joystick", ""},
	{"D-pad", "dpad", ""},
	{"LPad", "left_trackpad", ""},
	{"A", "button_diamond", "button_a"},
	{"B", "button_diamond", "button_b"},
	{"X", "button_diamond", "button_x"},
	{"Y", "button_diamond", "button_y"},
	{"Menu", "switch", "button_menu"},
	{"RPad", "right_trackpad", ""},
	{"RS", "right_joystick", ""},
	{"R4", "switch", "button_back_right_upper"},
	{"R5", "switch", "button_back_right"},
	{"R1", "switch", "right_bumper"},
	{"R2", "right_trigger", ""},
}

// Sheet describes the current mapping of every bound control.
func (e *Engine) Sheet() Sheet {
	st := e.State()
	sh := Sheet{Set: st.Set, Layers: st.Layers}
	if e.base == nil {
		return sh
	}
	for _, l := range layout {
		_, shifted := e.modeShift[l.source]
		c := e.control(l.glyph, l.source, l.input, e.layers, shifted)
		if c == nil {
			continue
		}
		if len(e.layers) > 0 || shifted {
			bare := e.control(l.glyph, l.source, l.input, nil, false)
			c.Changed = bare == nil || bare.Mode != c.Mode || !reflect.DeepEqual(bare.Entries, c.Entries)
		}
		sh.Controls = append(sh.Controls, *c)
	}
	return sh
}

func (e *Engine) control(glyph, source, input string, layers []*steamcfg.Set, shifted bool) *Control {
	g := e.cfg.ActiveGroup(e.base, layers, shifted, source)
	if g == nil {
		return nil
	}
	c := &Control{Glyph: glyph, Source: source, Mode: g.Mode}
	if input != "" {
		c.Entries = entries("", e.cfg.Activators(e.base, layers, shifted, source, input))
		if len(c.Entries) == 0 {
			return nil
		}
		return c
	}
	var inputs []string
	for in := range g.Inputs {
		inputs = append(inputs, in)
	}
	slices.SortFunc(inputs, func(a, b string) int {
		if d := inputRank(a) - inputRank(b); d != 0 {
			return d
		}
		return strings.Compare(a, b)
	})
	for _, in := range inputs {
		c.Entries = append(c.Entries, entries(in, e.cfg.Activators(e.base, layers, shifted, source, in))...)
	}
	return c
}

func entries(input string, acts []steamcfg.Activator) []Entry {
	var out []Entry
	for _, a := range acts {
		out = append(out, Entry{Input: input, Activator: a.Kind, Label: binding.Label(a.Bindings), Keys: binding.Chord(a.Bindings)})
	}
	return out
}

var fixedRank = map[string]int{"dpad_north": 0, "dpad_east": 1, "dpad_south": 2, "dpad_west": 3, "click": 4, "edge": 5, "touch": 6}

// inputRank orders inputs: directions clockwise, then clicks, then menu items
// by number, then everything else alphabetically.
func inputRank(in string) int {
	if r, ok := fixedRank[in]; ok {
		return r
	}
	if n, ok := strings.CutPrefix(in, "touch_menu_button_"); ok {
		if i, err := strconv.Atoi(n); err == nil {
			return 10 + i
		}
	}
	return 1000
}

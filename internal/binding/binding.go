// Package binding parses Steam Input binding strings such as
// "key_press LEFT_CONTROL, , ghost_045_move_0010.png, " and turns them into
// human-readable labels.
//
// A binding string is comma separated: the command and its space-separated
// arguments, then an optional user label, then an optional icon file name.
package binding

import (
	"strconv"
	"strings"
)

// Kind classifies a binding by its command.
type Kind int

const (
	Unknown Kind = iota
	KeyPress
	MouseButton
	MouseWheel
	XInputButton
	GameAction
	ControllerAction
	ModeShift
)

var kinds = map[string]Kind{
	"key_press":         KeyPress,
	"mouse_button":      MouseButton,
	"mouse_wheel":       MouseWheel,
	"xinput_button":     XInputButton,
	"game_action":       GameAction,
	"controller_action": ControllerAction,
	"mode_shift":        ModeShift,
}

// Binding is one parsed binding string.
type Binding struct {
	Kind    Kind
	Raw     string
	Command string
	Args    []string
	Label   string // user-provided label; may be a "#Localization" token
	Icon    string // binding_icons file name, if any
}

// Parse never fails: unrecognised commands become Kind Unknown so callers can
// still show the raw text.
func Parse(raw string) Binding {
	b := Binding{Raw: raw}
	fields := strings.Split(raw, ",")
	words := strings.Fields(fields[0])
	if len(words) > 0 {
		b.Command = words[0]
		b.Kind = kinds[b.Command]
		if len(words) > 1 {
			b.Args = words[1:]
		}
	}
	if len(fields) > 1 {
		b.Label = strings.TrimSpace(fields[1])
	}
	if len(fields) > 2 {
		b.Icon = strings.TrimSpace(fields[2])
	}
	return b
}

// Op is an operation on Steam Input's action-set/layer state.
type Op int

const (
	NoOp Op = iota
	AddLayer
	HoldLayer
	RemoveLayer
	ChangePreset // switch action set
)

var ops = map[string]Op{
	"add_layer":     AddLayer,
	"hold_layer":    HoldLayer,
	"remove_layer":  RemoveLayer,
	"change_preset": ChangePreset,
}

// StateChange describes a binding that changes the active action set or layers.
// Target is the 1-based index Steam writes in the binding.
type StateChange struct {
	Op     Op
	Target int
}

// StateChange reports whether b changes action set/layer state.
func (b Binding) StateChange() (StateChange, bool) {
	if b.Kind != ControllerAction || len(b.Args) < 2 {
		return StateChange{}, false
	}
	op, ok := ops[strings.ToLower(b.Args[0])]
	if !ok {
		return StateChange{}, false
	}
	n, err := strconv.Atoi(b.Args[1])
	if err != nil {
		return StateChange{}, false
	}
	return StateChange{Op: op, Target: n}, true
}

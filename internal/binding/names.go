package binding

import (
	"slices"
	"strconv"
	"strings"
)

var keyNames = map[string]string{
	"LEFT_CONTROL": "Ctrl", "RIGHT_CONTROL": "Ctrl",
	"LEFT_SHIFT": "Shift", "RIGHT_SHIFT": "Shift",
	"LEFT_ALT": "Alt", "RIGHT_ALT": "Alt",
	"LEFT_WINDOWS": "Meta", "RIGHT_WINDOWS": "Meta",
	"RETURN": "Enter", "ENTER": "Enter", "ESCAPE": "Esc", "SPACE": "Space",
	"BACKSPACE": "Backspace", "DELETE": "Del", "INSERT": "Ins", "TAB": "Tab",
	"HOME": "Home", "END": "End", "PAGE_UP": "PgUp", "PAGE_DOWN": "PgDn",
	"UP_ARROW": "↑", "DOWN_ARROW": "↓", "LEFT_ARROW": "←", "RIGHT_ARROW": "→",
	"DASH": "-", "EQUALS": "=", "PERIOD": ".", "BACK_TICK": "`", "COMMA": ",",
	"FORWARD_SLASH": "/", "SEMICOLON": ";", "SINGLE_QUOTE": "'",
	"LEFT_BRACKET": "[", "RIGHT_BRACKET": "]", "BACKSLASH": "\\",
	"VOLUME_UP": "Vol+", "VOLUME_DOWN": "Vol−", "MUTE": "Mute", "PLAY": "Play/Pause",
	"NEXT_TRACK": "Next", "PREV_TRACK": "Prev",
}

// modifierRank orders modifiers the way KDE displays shortcuts.
var modifierRank = map[string]int{"Meta": 0, "Ctrl": 1, "Alt": 2, "Shift": 3}

var mouseNames = map[string]string{
	"LEFT": "Left Click", "RIGHT": "Right Click", "MIDDLE": "Middle Click",
	"BACK": "Mouse Back", "FORWARD": "Mouse Forward",
	"SCROLL_UP": "Scroll ↑", "SCROLL_DOWN": "Scroll ↓",
	"SCROLL_LEFT": "Scroll ←", "SCROLL_RIGHT": "Scroll →",
}

var padNames = map[string]string{
	"DPAD_UP": "↑", "DPAD_DOWN": "↓", "DPAD_LEFT": "←", "DPAD_RIGHT": "→",
	"SHOULDER_LEFT": "LB", "SHOULDER_RIGHT": "RB",
	"TRIGGER_LEFT": "LT", "TRIGGER_RIGHT": "RT",
	"JOYSTICK_LEFT": "LS", "JOYSTICK_RIGHT": "RS",
	"START": "Start", "SELECT": "Back", "BACK": "Back",
}

// KeyName turns a Steam key_press argument into a short display name.
func KeyName(key string) string {
	if n, ok := keyNames[key]; ok {
		return n
	}
	if num, ok := strings.CutPrefix(key, "KEYPAD_"); ok {
		return "Num " + titleCase(num)
	}
	if len(key) <= 3 { // letters, digits, F1-F12
		return key
	}
	return titleCase(key)
}

// IsModifier reports whether key is Ctrl/Shift/Alt/Meta on either side.
func IsModifier(key string) bool {
	_, ok := modifierRank[keyNames[key]]
	return ok
}

// Chord renders the bindings fired together by one activator as display parts,
// e.g. ["Meta", "Shift", "."]. Modifiers come first in KDE order and repeat
// modifiers collapse; everything else keeps config order.
func Chord(bs []Binding) []string {
	var mods, rest []string
	seen := map[string]bool{}
	for _, b := range bs {
		if b.Kind == KeyPress && len(b.Args) > 0 && IsModifier(b.Args[0]) {
			m := KeyName(b.Args[0])
			if !seen[m] {
				seen[m] = true
				mods = append(mods, m)
			}
			continue
		}
		if s := Describe(b); s != "" {
			rest = append(rest, s)
		}
	}
	slices.SortStableFunc(mods, func(a, b string) int { return modifierRank[a] - modifierRank[b] })
	return append(mods, rest...)
}

// Label returns the first user label among bs, ignoring "#Localization"
// tokens. Steam stores an activator's label on one of its bindings.
func Label(bs []Binding) string {
	for _, b := range bs {
		if userLabel(b) {
			return b.Label
		}
	}
	return ""
}

func userLabel(b Binding) bool { return b.Label != "" && !strings.HasPrefix(b.Label, "#") }

// Describe renders a single binding. Physical outputs (keys, mouse, pad) are
// always named by what they send; other bindings prefer a user label.
func Describe(b Binding) string {
	physical := b.Kind == KeyPress || b.Kind == MouseButton || b.Kind == MouseWheel || b.Kind == XInputButton
	if !physical && userLabel(b) {
		return b.Label
	}
	arg := ""
	if len(b.Args) > 0 {
		arg = b.Args[len(b.Args)-1]
	}
	switch b.Kind {
	case KeyPress:
		return KeyName(arg)
	case MouseButton, MouseWheel:
		if n, ok := mouseNames[strings.ToUpper(arg)]; ok {
			return n
		}
		return titleCase(arg)
	case XInputButton:
		if n, ok := padNames[strings.ToUpper(arg)]; ok {
			return "Pad " + n
		}
		return "Pad " + strings.ToUpper(arg)
	case GameAction:
		return titleCase(arg)
	case ControllerAction:
		if sc, ok := b.StateChange(); ok {
			return stateChangeName(sc)
		}
		if len(b.Args) > 0 {
			return titleCase(b.Args[0])
		}
	case ModeShift:
		return ""
	}
	return strings.TrimSpace(strings.Split(b.Raw, ",")[0])
}

func stateChangeName(sc StateChange) string {
	n := strconv.Itoa(sc.Target)
	switch sc.Op {
	case AddLayer:
		return "Layer " + n + " on"
	case HoldLayer:
		return "Hold layer " + n
	case RemoveLayer:
		return "Layer " + n + " off"
	case ChangePreset:
		return "Action set " + n
	}
	return ""
}

func titleCase(s string) string {
	words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return r == '_' || r == ' ' })
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

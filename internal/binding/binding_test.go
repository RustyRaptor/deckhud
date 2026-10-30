package binding

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		raw  string
		want Binding
	}{
		{"key_press LEFT_CONTROL, , ghost_045_move_0010.png, ", Binding{
			Kind: KeyPress, Command: "key_press", Args: []string{"LEFT_CONTROL"}, Icon: "ghost_045_move_0010.png"}},
		{"key_press F11, , ", Binding{Kind: KeyPress, Command: "key_press", Args: []string{"F11"}}},
		{"key_press V", Binding{Kind: KeyPress, Command: "key_press", Args: []string{"V"}}},
		{"mouse_button LEFT, , ", Binding{Kind: MouseButton, Command: "mouse_button", Args: []string{"LEFT"}}},
		{"mouse_wheel SCROLL_UP, , ", Binding{Kind: MouseWheel, Command: "mouse_wheel", Args: []string{"SCROLL_UP"}}},
		{"xinput_button dpad_up, , ", Binding{Kind: XInputButton, Command: "xinput_button", Args: []string{"dpad_up"}}},
		{"game_action Default B, Back, , ", Binding{
			Kind: GameAction, Command: "game_action", Args: []string{"Default", "B"}, Label: "Back"}},
		{"controller_action hold_layer 2 1 0, Scroll (Hold), , ", Binding{
			Kind: ControllerAction, Command: "controller_action", Args: []string{"hold_layer", "2", "1", "0"}, Label: "Scroll (Hold)"}},
		{"controller_action SHOW_KEYBOARD, ⌨️, , ", Binding{
			Kind: ControllerAction, Command: "controller_action", Args: []string{"SHOW_KEYBOARD"}, Label: "⌨️"}},
		{"mode_shift right_joystick 37", Binding{Kind: ModeShift, Command: "mode_shift", Args: []string{"right_joystick", "37"}}},
		{"some_future_thing 1 2, Hi, , ", Binding{Kind: Unknown, Command: "some_future_thing", Args: []string{"1", "2"}, Label: "Hi"}},
	}
	for _, c := range cases {
		t.Run(c.raw, func(t *testing.T) {
			c.want.Raw = c.raw
			if got := Parse(c.raw); !reflect.DeepEqual(got, c.want) {
				t.Errorf("Parse(%q)\n got  %+v\n want %+v", c.raw, got, c.want)
			}
		})
	}
}

func TestStateChange(t *testing.T) {
	cases := []struct {
		raw    string
		want   StateChange
		wantOK bool
	}{
		{"controller_action hold_layer 2 1 0, Scroll (Hold), , ", StateChange{Op: HoldLayer, Target: 2}, true},
		{"controller_action add_layer 3 0 0", StateChange{Op: AddLayer, Target: 3}, true},
		{"controller_action remove_layer 3 0 0", StateChange{Op: RemoveLayer, Target: 3}, true},
		{"controller_action CHANGE_PRESET 2 1 1, , ", StateChange{Op: ChangePreset, Target: 2}, true},
		{"controller_action SCREENSHOT, , ", StateChange{}, false},
		{"controller_action hold_layer x", StateChange{}, false},
		{"key_press A", StateChange{}, false},
	}
	for _, c := range cases {
		got, ok := Parse(c.raw).StateChange()
		if ok != c.wantOK || got != c.want {
			t.Errorf("%q: got %+v,%v want %+v,%v", c.raw, got, ok, c.want, c.wantOK)
		}
	}
}

func TestKeyName(t *testing.T) {
	cases := map[string]string{
		"LEFT_CONTROL": "Ctrl", "RIGHT_CONTROL": "Ctrl",
		"LEFT_SHIFT": "Shift", "RIGHT_SHIFT": "Shift",
		"LEFT_ALT": "Alt", "RIGHT_ALT": "Alt",
		"LEFT_WINDOWS": "Meta", "RIGHT_WINDOWS": "Meta",
		"RETURN": "Enter", "ENTER": "Enter", "ESCAPE": "Esc", "SPACE": "Space",
		"BACKSPACE": "Backspace", "DELETE": "Del", "TAB": "Tab",
		"UP_ARROW": "↑", "DOWN_ARROW": "↓", "LEFT_ARROW": "←", "RIGHT_ARROW": "→",
		"PAGE_UP": "PgUp", "PAGE_DOWN": "PgDn",
		"DASH": "-", "EQUALS": "=", "PERIOD": ".", "BACK_TICK": "`", "COMMA": ",",
		"FORWARD_SLASH": "/", "SEMICOLON": ";", "SINGLE_QUOTE": "'",
		"LEFT_BRACKET": "[", "RIGHT_BRACKET": "]", "BACKSLASH": "\\",
		"KEYPAD_1": "Num 1", "F11": "F11", "A": "A", "7": "7",
		"VOLUME_UP": "Vol+", "VOLUME_DOWN": "Vol−", "PLAY": "Play/Pause",
		"NEXT_TRACK": "Next", "PREV_TRACK": "Prev",
		"SOME_NEW_KEY": "Some New Key",
	}
	for in, want := range cases {
		if got := KeyName(in); got != want {
			t.Errorf("KeyName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsModifier(t *testing.T) {
	for _, k := range []string{"LEFT_CONTROL", "RIGHT_SHIFT", "LEFT_ALT", "LEFT_WINDOWS"} {
		if !IsModifier(k) {
			t.Errorf("%s should be a modifier", k)
		}
	}
	for _, k := range []string{"A", "SPACE", "RETURN"} {
		if IsModifier(k) {
			t.Errorf("%s should not be a modifier", k)
		}
	}
}

func TestChord(t *testing.T) {
	parse := func(raws ...string) []Binding {
		var bs []Binding
		for _, r := range raws {
			bs = append(bs, Parse(r))
		}
		return bs
	}
	cases := []struct {
		name string
		in   []Binding
		want []string
	}{
		{"single key", parse("key_press F10, , "), []string{"F10"}},
		{"ctrl+v", parse("key_press LEFT_CONTROL, , ", "key_press V, , "), []string{"Ctrl", "V"}},
		{"modifiers sorted Meta Ctrl Alt Shift, keys keep order",
			parse("key_press RIGHT_SHIFT, , ", "key_press LEFT_WINDOWS, , ", "key_press PERIOD, , "),
			[]string{"Meta", "Shift", "."}},
		{"duplicate modifiers collapse",
			parse("key_press LEFT_CONTROL", "key_press RIGHT_CONTROL", "key_press K"),
			[]string{"Ctrl", "K"}},
		{"mouse and wheel", parse("mouse_button RIGHT, , ", "mouse_wheel SCROLL_UP"),
			[]string{"Right Click", "Scroll ↑"}},
		{"gamepad", parse("xinput_button A, , ", "xinput_button dpad_up"), []string{"Pad A", "Pad ↑"}},
		{"label wins for actions", parse("controller_action SHOW_KEYBOARD, ⌨️, , "), []string{"⌨️"}},
		{"action without label", parse("controller_action SCREENSHOT, , "), []string{"Screenshot"}},
		{"empty", nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Chord(c.in); !reflect.DeepEqual(got, c.want) {
				t.Errorf("Chord = %q, want %q", got, c.want)
			}
		})
	}
}

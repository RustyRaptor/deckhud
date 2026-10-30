package hid

import (
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"
)

// idle is a real report captured from a Steam Deck (LCD) with Steam running.
const idle = "0100094059 5f0e01 0000000000000000 0000000000000000 0000000000000000 0000000000000000" +
	"0000000000000000 aa01cbfdadffc5ff 0000000008000200"

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func blank() []byte {
	b := make([]byte, ReportSize)
	b[0], b[1], b[2], b[3] = 0x01, 0x00, 0x09, 0x40
	return b
}

func TestDecodeRealIdleReport(t *testing.T) {
	s, err := Decode(mustHex(t, idle))
	if err != nil {
		t.Fatal(err)
	}
	if s.Seq != 0x010e5f59 {
		t.Errorf("Seq = %#x", s.Seq)
	}
	if s.Buttons != 0 {
		t.Errorf("Buttons = %v, want none", s.Buttons)
	}
	if s.LeftStick != (Vec{426, -565}) || s.RightStick != (Vec{-83, -59}) {
		t.Errorf("sticks = %v %v", s.LeftStick, s.RightStick)
	}
}

func TestDecodeButtonBits(t *testing.T) {
	cases := []struct {
		byteIdx, bit int
		want         Button
	}{
		{8, 0, R2}, {8, 1, L2}, {8, 2, R1}, {8, 3, L1},
		{8, 4, Y}, {8, 5, B}, {8, 6, X}, {8, 7, A},
		{9, 0, DpadUp}, {9, 1, DpadRight}, {9, 2, DpadLeft}, {9, 3, DpadDown},
		{9, 4, View}, {9, 5, Steam}, {9, 6, Menu}, {9, 7, L5},
		{10, 0, R5}, {10, 1, LeftPadClick}, {10, 2, RightPadClick},
		{10, 3, LeftPadTouch}, {10, 4, RightPadTouch}, {10, 6, L3},
		{11, 2, R3},
		{13, 1, L4}, {13, 2, R4}, {13, 6, LeftStickTouch}, {13, 7, RightStickTouch},
		{14, 2, QAM},
	}
	for _, c := range cases {
		r := blank()
		r[c.byteIdx] = 1 << c.bit
		s, err := Decode(r)
		if err != nil {
			t.Fatal(err)
		}
		if s.Buttons != c.want {
			t.Errorf("byte %d bit %d: got %v, want %v", c.byteIdx, c.bit, s.Buttons, c.want)
		}
	}
}

func TestDecodeAxes(t *testing.T) {
	r := blank()
	put := func(off int, v int16) { binary.LittleEndian.PutUint16(r[off:], uint16(v)) }
	put(16, -100)
	put(18, 200)
	put(20, 300)
	put(22, -400)
	put(44, 32767)
	put(46, 1000)
	put(48, -32767)
	put(50, 32767)
	put(52, 12345)
	put(54, -12345)
	put(56, 50)
	put(58, 60)
	s, err := Decode(r)
	if err != nil {
		t.Fatal(err)
	}
	want := State{
		LeftPad: Vec{-100, 200}, RightPad: Vec{300, -400},
		LeftTrigger: 32767, RightTrigger: 1000,
		LeftStick: Vec{-32767, 32767}, RightStick: Vec{12345, -12345},
		LeftPadPressure: 50, RightPadPressure: 60,
	}
	if s != want {
		t.Errorf("got  %+v\nwant %+v", s, want)
	}
}

func TestDecodeRejectsOtherReports(t *testing.T) {
	short := blank()[:20]
	other := blank()
	other[2] = 0x04 // battery/status report
	for name, r := range map[string][]byte{"short": short, "other type": other} {
		if _, err := Decode(r); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	if _, err := Decode(other); err != ErrNotInput {
		t.Errorf("other type should return ErrNotInput, got %v", err)
	}
}

func TestButtonSetHelpers(t *testing.T) {
	set := A | L4 | QAM
	if !set.Has(A) || !set.Has(L4) || set.Has(B) {
		t.Error("Has wrong")
	}
	if got := set.String(); got != "A+L4+QAM" {
		t.Errorf("String = %q", got)
	}
	pressed, released := (A | B).Diff(B | X)
	if pressed != X || released != A {
		t.Errorf("Diff: pressed %v released %v", pressed, released)
	}
	if got := Button(0).String(); got != "none" {
		t.Errorf("empty String = %q", got)
	}
}

func TestSteamInputNames(t *testing.T) {
	// Source (group_source_bindings key) and input name used in Steam configs.
	cases := map[Button]Input{
		A: {"button_diamond", "button_a"}, B: {"button_diamond", "button_b"},
		X: {"button_diamond", "button_x"}, Y: {"button_diamond", "button_y"},
		L1: {"switch", "left_bumper"}, R1: {"switch", "right_bumper"},
		View: {"switch", "button_escape"}, Menu: {"switch", "button_menu"},
		L5: {"switch", "button_back_left"}, R5: {"switch", "button_back_right"},
		L4: {"switch", "button_back_left_upper"}, R4: {"switch", "button_back_right_upper"},
		DpadUp: {"dpad", "dpad_north"}, DpadDown: {"dpad", "dpad_south"},
		DpadLeft: {"dpad", "dpad_west"}, DpadRight: {"dpad", "dpad_east"},
		L3: {"joystick", "click"}, R3: {"right_joystick", "click"},
		L2: {"left_trigger", "click"}, R2: {"right_trigger", "click"},
		LeftPadClick: {"left_trackpad", "click"}, RightPadClick: {"right_trackpad", "click"},
	}
	for b, want := range cases {
		if got, ok := b.SteamInput(); !ok || got != want {
			t.Errorf("%v.SteamInput() = %v,%v want %v", b, got, ok, want)
		}
	}
	for _, b := range []Button{Steam, QAM, LeftPadTouch, A | B} {
		if _, ok := b.SteamInput(); ok {
			t.Errorf("%v should have no Steam input mapping", b)
		}
	}
}

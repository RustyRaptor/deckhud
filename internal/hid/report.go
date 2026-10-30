// Package hid decodes the Steam Deck ("Neptune") controller's raw HID input
// reports, which can be read from /dev/hidrawN alongside Steam without
// disturbing it.
//
// Layout follows the Linux hid-steam driver (steam_do_deck_input_event): a
// 4-byte header (version u16, type u8 = 0x09, length u8 = 0x40), then a
// little-endian payload.
package hid

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"strings"
)

// ReportSize is the size of one Deck input report in bytes.
const ReportSize = 64

const typeDeckState = 0x09

// ErrNotInput is returned for well-formed reports that aren't controller state
// (battery/status reports share the same endpoint).
var ErrNotInput = errors.New("hid: not a deck state report")

// Vec is a raw signed 2D axis. Y is positive up for sticks and pads.
type Vec struct{ X, Y int16 }

// State is one decoded controller snapshot.
type State struct {
	Seq                               uint32
	Buttons                           Button
	LeftPad, RightPad                 Vec
	LeftTrigger, RightTrigger         uint16 // 0..32767
	LeftStick, RightStick             Vec
	LeftPadPressure, RightPadPressure uint16
}

// Decode parses a single report.
func Decode(r []byte) (State, error) {
	if len(r) < 60 {
		return State{}, fmt.Errorf("hid: short report (%d bytes)", len(r))
	}
	if r[2] != typeDeckState {
		return State{}, ErrNotInput
	}
	le := binary.LittleEndian
	i16 := func(off int) int16 { return int16(le.Uint16(r[off:])) }
	return State{
		Seq:              le.Uint32(r[4:]),
		Buttons:          Button(le.Uint64(r[8:])) & allButtons,
		LeftPad:          Vec{i16(16), i16(18)},
		RightPad:         Vec{i16(20), i16(22)},
		LeftTrigger:      le.Uint16(r[44:]),
		RightTrigger:     le.Uint16(r[46:]),
		LeftStick:        Vec{i16(48), i16(50)},
		RightStick:       Vec{i16(52), i16(54)},
		LeftPadPressure:  le.Uint16(r[56:]),
		RightPadPressure: le.Uint16(r[58:]),
	}, nil
}

// Button is a bit set of physical controls. Bit n is bit (n%8) of report byte
// 8+n/8.
type Button uint64

const (
	R2              Button = 1 << 0 // digital full pull
	L2              Button = 1 << 1
	R1              Button = 1 << 2
	L1              Button = 1 << 3
	Y               Button = 1 << 4
	B               Button = 1 << 5
	X               Button = 1 << 6
	A               Button = 1 << 7
	DpadUp          Button = 1 << 8
	DpadRight       Button = 1 << 9
	DpadLeft        Button = 1 << 10
	DpadDown        Button = 1 << 11
	View            Button = 1 << 12
	Steam           Button = 1 << 13
	Menu            Button = 1 << 14
	L5              Button = 1 << 15 // lower left back button
	R5              Button = 1 << 16 // lower right back button
	LeftPadClick    Button = 1 << 17
	RightPadClick   Button = 1 << 18
	LeftPadTouch    Button = 1 << 19
	RightPadTouch   Button = 1 << 20
	L3              Button = 1 << 22
	R3              Button = 1 << 26
	L4              Button = 1 << 41 // upper left back button
	R4              Button = 1 << 42 // upper right back button
	LeftStickTouch  Button = 1 << 46
	RightStickTouch Button = 1 << 47
	QAM             Button = 1 << 50 // quick access ("...") button
)

var buttonNames = []struct {
	b    Button
	name string
}{
	{A, "A"}, {B, "B"}, {X, "X"}, {Y, "Y"},
	{L1, "L1"}, {R1, "R1"}, {L2, "L2"}, {R2, "R2"}, {L3, "L3"}, {R3, "R3"},
	{L4, "L4"}, {R4, "R4"}, {L5, "L5"}, {R5, "R5"},
	{DpadUp, "Up"}, {DpadDown, "Down"}, {DpadLeft, "Left"}, {DpadRight, "Right"},
	{View, "View"}, {Menu, "Menu"}, {Steam, "Steam"}, {QAM, "QAM"},
	{LeftPadClick, "LPadClick"}, {RightPadClick, "RPadClick"},
	{LeftPadTouch, "LPadTouch"}, {RightPadTouch, "RPadTouch"},
	{LeftStickTouch, "LStickTouch"}, {RightStickTouch, "RStickTouch"},
}

var allButtons = func() Button {
	var all Button
	for _, n := range buttonNames {
		all |= n.b
	}
	return all
}()

// Has reports whether every button in o is in b.
func (b Button) Has(o Button) bool { return b&o == o }

// Diff returns the buttons newly pressed and newly released going from b to next.
func (b Button) Diff(next Button) (pressed, released Button) {
	return next &^ b, b &^ next
}

// Each calls f for every single button in b, in stable order.
func (b Button) Each(f func(Button)) {
	for _, n := range buttonNames {
		if b.Has(n.b) {
			f(n.b)
		}
	}
}

func (b Button) String() string {
	if b == 0 {
		return "none"
	}
	var parts []string
	b.Each(func(o Button) {
		for _, n := range buttonNames {
			if n.b == o {
				parts = append(parts, n.name)
			}
		}
	})
	return strings.Join(parts, "+")
}

// Input names a control the way Steam Input configs do: Source is the key used
// in a preset's group_source_bindings, Name is the input inside the group.
type Input struct {
	Source string
	Name   string
}

var steamInputs = map[Button]Input{
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

// SteamInput maps a single button to its Steam config source and input name.
func (b Button) SteamInput() (Input, bool) {
	if bits.OnesCount64(uint64(b)) != 1 {
		return Input{}, false
	}
	in, ok := steamInputs[b]
	return in, ok
}

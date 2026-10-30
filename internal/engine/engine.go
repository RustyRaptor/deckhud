// Package engine replays the parts of Steam Input that the HUD needs to show:
// which activator fired for a button, which keys that sent, the action-set and
// layer stack, and radial menu selection.
//
// It is a pure state machine: feed it controller states with timestamps and it
// returns events. It never touches devices or the clock itself.
package engine

import (
	"math"
	"strconv"
	"time"

	"github.com/RustyRaptor/deckhud/internal/binding"
	"github.com/RustyRaptor/deckhud/internal/hid"
	"github.com/RustyRaptor/deckhud/internal/steamcfg"
)

// Event is one thing the HUD should show. Exactly one payload is set.
type Event struct {
	Type   string    `json:"type"` // "chord", "radial" or "state"
	Chord  *Chord    `json:"chord,omitempty"`
	Radial *Radial   `json:"radial,omitempty"`
	State  *SetState `json:"state,omitempty"`
}

// Chord reports a button activator that sent output (keys, mouse, pad, ...).
type Chord struct {
	Button    string   `json:"button"` // physical control, e.g. "R4"
	Input     string   `json:"input"`  // Steam input name
	Activator string   `json:"activator"`
	Label     string   `json:"label,omitempty"`
	Keys      []string `json:"keys"`
}

// Radial reports radial menu activity. Phase is open, move, fire or cancel.
type Radial struct {
	Phase  string     `json:"phase"`
	Source string     `json:"source"`
	Title  string     `json:"title,omitempty"`
	Items  []MenuItem `json:"items,omitempty"`
	Index  int        `json:"index"`
	Keys   []string   `json:"keys,omitempty"` // on fire
}

// MenuItem is one radial entry as displayed.
type MenuItem struct {
	Label string   `json:"label,omitempty"`
	Icon  string   `json:"icon,omitempty"`
	Keys  []string `json:"keys"`
}

// SetState is the active action set and layer stack (bottom first).
type SetState struct {
	Set    string   `json:"set"`
	Layers []string `json:"layers,omitempty"`
}

// Options tunes timing and thresholds. Zero values pick defaults.
type Options struct {
	LongPress   time.Duration // default 300ms, overridden by long_press_time
	DoublePress time.Duration // default 250ms
	OpenRadius  float64       // stick deflection (0..1) that opens a menu; default 0.4
	FireRadius  float64       // deflection below which the menu fires; default 0.25
}

// Engine is the Steam Input replica. Not safe for concurrent use.
type Engine struct {
	cfg    *steamcfg.Config
	opts   Options
	base   *steamcfg.Set
	layers []*steamcfg.Set

	prev      hid.Button
	buttons   map[hid.Button]*buttonState
	modeShift map[string]hid.Button // source -> button holding the shift
	radials   map[string]*radialState
}

type buttonState struct {
	acts        []steamcfg.Activator // snapshot at press
	input       hid.Input
	pressedAt   time.Time
	releasedAt  time.Time
	held        bool
	longArmed   bool
	longFired   bool
	pendingFull bool // a Full_Press waiting on long/double resolution
	awaitDouble bool // released; Full fires unless pressed again in time
	heldLayers  []*steamcfg.Set
}

type radialState struct {
	group *steamcfg.Group
	index int
}

// New creates an engine positioned on the config's first action set.
func New(cfg *steamcfg.Config, opts Options) *Engine {
	if opts.LongPress == 0 {
		opts.LongPress = 300 * time.Millisecond
	}
	if opts.DoublePress == 0 {
		opts.DoublePress = 250 * time.Millisecond
	}
	if opts.OpenRadius == 0 {
		opts.OpenRadius = 0.4
	}
	if opts.FireRadius == 0 {
		opts.FireRadius = 0.25
	}
	return &Engine{
		cfg:       cfg,
		opts:      opts,
		base:      cfg.Base(),
		buttons:   map[hid.Button]*buttonState{},
		modeShift: map[string]hid.Button{},
		radials:   map[string]*radialState{},
	}
}

// State returns the current action set and layers.
func (e *Engine) State() SetState {
	st := SetState{}
	if e.base != nil {
		st.Set = e.base.Title
	}
	for _, l := range e.layers {
		st.Layers = append(st.Layers, l.Title)
	}
	return st
}

// Update consumes one controller state.
func (e *Engine) Update(s hid.State, now time.Time) []Event {
	var out []Event
	pressed, released := e.prev.Diff(s.Buttons)
	e.prev = s.Buttons

	released.Each(func(b hid.Button) { out = append(out, e.release(b, now)...) })
	pressed.Each(func(b hid.Button) { out = append(out, e.press(b, now)...) })
	out = append(out, e.timers(now)...)

	out = append(out, e.radial("right_joystick", s.RightStick)...)
	out = append(out, e.radial("joystick", s.LeftStick)...)
	return out
}

func (e *Engine) activators(in hid.Input) []steamcfg.Activator {
	if e.base == nil {
		return nil
	}
	_, shifted := e.modeShift[in.Source]
	return e.cfg.Activators(e.base, e.layers, shifted, in.Source, in.Name)
}

func find(acts []steamcfg.Activator, kind string) *steamcfg.Activator {
	for i := range acts {
		if acts[i].Kind == kind {
			return &acts[i]
		}
	}
	return nil
}

func (e *Engine) press(b hid.Button, now time.Time) []Event {
	in, ok := b.SteamInput()
	if !ok {
		return nil
	}
	bs := e.buttons[b]
	if bs == nil {
		bs = &buttonState{}
		e.buttons[b] = bs
	}
	var out []Event

	if bs.awaitDouble && now.Sub(bs.releasedAt) <= e.opts.DoublePress {
		// Second press in time: the double press replaces the pending single.
		bs.awaitDouble = false
		bs.held, bs.pressedAt = true, now
		if a := find(bs.acts, "Double_Press"); a != nil {
			out = append(out, e.fire(b, bs, *a)...)
		}
		bs.pendingFull, bs.longArmed = false, false
		return out
	}
	if bs.awaitDouble { // window expired between timer ticks
		out = append(out, e.fireFull(b, bs)...)
	}

	acts := e.activators(in)
	*bs = buttonState{acts: acts, input: in, pressedAt: now, held: true}
	if len(acts) == 0 {
		return out
	}
	if a := find(acts, "Start_Press"); a != nil {
		out = append(out, e.fire(b, bs, *a)...)
	}
	hasLong := find(acts, "Long_Press") != nil
	hasDouble := find(acts, "Double_Press") != nil
	bs.longArmed = hasLong
	if find(acts, "Full_Press") != nil {
		if hasLong || hasDouble {
			bs.pendingFull = true
		} else {
			out = append(out, e.fireFull(b, bs)...)
		}
	}
	return out
}

func (e *Engine) release(b hid.Button, now time.Time) []Event {
	bs := e.buttons[b]
	if bs == nil || !bs.held {
		return nil
	}
	var out []Event
	bs.held = false
	bs.releasedAt = now
	bs.longArmed = false
	if a := find(bs.acts, "Release_Press"); a != nil {
		out = append(out, e.fire(b, bs, *a)...)
	}
	if bs.pendingFull && !bs.longFired {
		if find(bs.acts, "Double_Press") != nil {
			bs.awaitDouble = true
		} else {
			out = append(out, e.fireFull(b, bs)...)
		}
	}
	bs.pendingFull = false

	for src, holder := range e.modeShift {
		if holder == b {
			delete(e.modeShift, src)
		}
	}
	if len(bs.heldLayers) > 0 {
		for _, l := range bs.heldLayers {
			e.removeLayer(l)
		}
		bs.heldLayers = nil
		out = append(out, e.stateEvent())
	}
	return out
}

func (e *Engine) timers(now time.Time) []Event {
	var out []Event
	var tracked hid.Button
	for b := range e.buttons {
		tracked |= b
	}
	tracked.Each(func(b hid.Button) { // stable order
		bs := e.buttons[b]
		if bs.held && bs.longArmed && now.Sub(bs.pressedAt) >= e.longPressFor(bs.acts) {
			bs.longArmed = false
			bs.longFired = true
			bs.pendingFull = false
			if a := find(bs.acts, "Long_Press"); a != nil {
				out = append(out, e.fire(b, bs, *a)...)
			}
		}
		if bs.awaitDouble && now.Sub(bs.releasedAt) > e.opts.DoublePress {
			out = append(out, e.fireFull(b, bs)...)
		}
	})
	return out
}

func (e *Engine) longPressFor(acts []steamcfg.Activator) time.Duration {
	if a := find(acts, "Long_Press"); a != nil {
		if ms, err := strconv.Atoi(a.Settings["long_press_time"]); err == nil && ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return e.opts.LongPress
}

func (e *Engine) fireFull(b hid.Button, bs *buttonState) []Event {
	bs.awaitDouble = false
	if a := find(bs.acts, "Full_Press"); a != nil {
		return e.fire(b, bs, *a)
	}
	return nil
}

// fire applies an activator: state changes and mode shifts take effect, and
// everything else is reported as a chord.
func (e *Engine) fire(b hid.Button, bs *buttonState, a steamcfg.Activator) []Event {
	var out []Event
	var outputs []binding.Binding
	stateChanged := false
	for _, bd := range a.Bindings {
		if sc, ok := bd.StateChange(); ok {
			if e.applyStateChange(sc, bs) {
				stateChanged = true
			}
			continue
		}
		if bd.Kind == binding.ModeShift {
			if len(bd.Args) > 0 && bs.held {
				e.modeShift[bd.Args[0]] = b
			}
			continue
		}
		outputs = append(outputs, bd)
	}
	if len(outputs) > 0 {
		out = append(out, Event{Type: "chord", Chord: &Chord{
			Button:    b.String(),
			Input:     bs.input.Name,
			Activator: a.Kind,
			Label:     binding.Label(outputs),
			Keys:      binding.Chord(outputs),
		}})
	}
	if stateChanged {
		out = append(out, e.stateEvent())
	}
	return out
}

func (e *Engine) applyStateChange(sc binding.StateChange, bs *buttonState) bool {
	target := e.cfg.SetByTarget(sc.Target)
	if target == nil {
		return false
	}
	switch sc.Op {
	case binding.ChangePreset:
		e.base, e.layers = target, nil
	case binding.AddLayer:
		e.removeLayer(target)
		e.layers = append(e.layers, target)
	case binding.HoldLayer:
		e.removeLayer(target)
		e.layers = append(e.layers, target)
		if bs.held {
			bs.heldLayers = append(bs.heldLayers, target)
		}
	case binding.RemoveLayer:
		e.removeLayer(target)
	default:
		return false
	}
	return true
}

func (e *Engine) removeLayer(l *steamcfg.Set) {
	for i, x := range e.layers {
		if x == l {
			e.layers = append(e.layers[:i:i], e.layers[i+1:]...)
			return
		}
	}
}

func (e *Engine) stateEvent() Event {
	st := e.State()
	return Event{Type: "state", State: &st}
}

// radial tracks a stick bound to a radial_menu group.
func (e *Engine) radial(source string, v hid.Vec) []Event {
	var g *steamcfg.Group
	if e.base != nil {
		_, shifted := e.modeShift[source]
		g = e.cfg.ActiveGroup(e.base, e.layers, shifted, source)
	}
	if g != nil && (g.Mode != "radial_menu" || len(g.MenuItems()) == 0) {
		g = nil
	}
	rs := e.radials[source]
	if rs != nil && rs.group != g {
		delete(e.radials, source)
		return []Event{{Type: "radial", Radial: &Radial{Phase: "cancel", Source: source, Index: rs.index}}}
	}
	if g == nil {
		return nil
	}

	mag := math.Hypot(float64(v.X), float64(v.Y)) / 32767
	items := g.MenuItems()
	switch {
	case rs == nil && mag >= e.opts.OpenRadius:
		idx := wedge(v, len(items))
		e.radials[source] = &radialState{group: g, index: idx}
		r := &Radial{Phase: "open", Source: source, Title: g.Name, Index: idx}
		for _, it := range items {
			r.Items = append(r.Items, MenuItem{Label: it.Label, Icon: it.Icon, Keys: binding.Chord(it.Bindings)})
		}
		return []Event{{Type: "radial", Radial: r}}
	case rs != nil && mag < e.opts.FireRadius:
		delete(e.radials, source)
		return []Event{{Type: "radial", Radial: &Radial{
			Phase: "fire", Source: source, Index: rs.index, Keys: binding.Chord(items[rs.index].Bindings)}}}
	case rs != nil && mag >= e.opts.OpenRadius:
		if idx := wedge(v, len(items)); idx != rs.index {
			rs.index = idx
			return []Event{{Type: "radial", Radial: &Radial{Phase: "move", Source: source, Index: idx}}}
		}
	}
	return nil
}

// wedge maps a direction to one of n equal wedges, item 0 centred at the top,
// counting clockwise.
func wedge(v hid.Vec, n int) int {
	deg := math.Atan2(float64(v.X), float64(v.Y)) * 180 / math.Pi
	w := 360 / float64(n)
	return int(math.Mod(deg+w/2+720, 360)/w) % n
}

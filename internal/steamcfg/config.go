// Package steamcfg models a Steam Input controller configuration
// ("controller_mappings" VDF) and answers "what does this control do right
// now?" for a given action set, layer stack and mode-shift state.
package steamcfg

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/RustyRaptor/deckhud/internal/binding"
	"github.com/RustyRaptor/deckhud/internal/vdf"
)

// Config is one parsed controller configuration.
type Config struct {
	Title  string
	Sets   []*Set // action sets and layers, ordered by preset id
	Groups map[int]*Group
}

// Set is an action set or, when Layer is true, an action layer on top of Parent.
type Set struct {
	ID      int // preset id; binding targets are ID+1
	Key     string
	Title   string
	Layer   bool
	Parent  string
	Sources []SourceBinding
}

// SourceBinding attaches a group to a physical source within a set.
type SourceBinding struct {
	GroupID   int
	Source    string // "switch", "button_diamond", "right_joystick", ...
	Active    bool
	ModeShift bool
}

// Group is a mode (dpad, radial_menu, switches, ...) with its inputs.
type Group struct {
	ID       int
	Mode     string
	Name     string
	Layer    bool
	Inputs   map[string][]Activator
	Settings map[string]string
}

// Activator is one way of triggering an input (Full_Press, Long_Press, ...).
type Activator struct {
	Kind     string
	Bindings []binding.Binding
	Settings map[string]string
}

// Load parses a controller_mappings document.
func Load(r io.Reader) (*Config, error) {
	root, err := vdf.Parse(r)
	if err != nil {
		return nil, err
	}
	cm := root.Get("controller_mappings")
	if cm == nil {
		return nil, fmt.Errorf("steamcfg: no controller_mappings block")
	}
	c := &Config{Title: cm.Str("title"), Groups: map[int]*Group{}}

	for _, gn := range cm.All("group") {
		g := parseGroup(gn)
		c.Groups[g.ID] = g
	}

	byKey := map[string]*Set{}
	for _, a := range cm.Get("actions").Children {
		s := &Set{ID: -1, Key: a.Key, Title: a.Str("title")}
		byKey[strings.ToLower(a.Key)] = s
		c.Sets = append(c.Sets, s)
	}
	for _, a := range cm.Get("action_layers").Children {
		s := &Set{ID: -1, Key: a.Key, Title: a.Str("title"), Layer: true, Parent: a.Str("parent_set_name")}
		byKey[strings.ToLower(a.Key)] = s
		c.Sets = append(c.Sets, s)
	}
	for _, p := range cm.All("preset") {
		s := byKey[strings.ToLower(p.Str("name"))]
		if s == nil {
			continue
		}
		s.ID, _ = strconv.Atoi(p.Str("id"))
		for _, sb := range p.Get("group_source_bindings").Children {
			id, err := strconv.Atoi(sb.Key)
			if err != nil {
				continue
			}
			f := strings.Fields(sb.Value)
			if len(f) == 0 {
				continue
			}
			s.Sources = append(s.Sources, SourceBinding{
				GroupID:   id,
				Source:    f[0],
				Active:    slices.Contains(f[1:], "active"),
				ModeShift: slices.Contains(f[1:], "modeshift"),
			})
		}
	}
	slices.SortStableFunc(c.Sets, func(a, b *Set) int { return a.ID - b.ID })
	return c, nil
}

func parseGroup(n *vdf.Node) *Group {
	id, _ := strconv.Atoi(n.Str("id"))
	g := &Group{
		ID:       id,
		Mode:     n.Str("mode"),
		Name:     n.Str("name"),
		Inputs:   map[string][]Activator{},
		Settings: leaves(n.Get("settings")),
	}
	g.Layer = g.Settings["layer"] == "1"
	for _, in := range n.Get("inputs").Children {
		var acts []Activator
		for _, a := range in.Get("activators").Children {
			act := Activator{Kind: a.Key, Settings: leaves(a.Get("settings"))}
			for _, b := range a.Get("bindings").All("binding") {
				act.Bindings = append(act.Bindings, binding.Parse(b.Value))
			}
			acts = append(acts, act)
		}
		if len(acts) > 0 {
			g.Inputs[in.Key] = acts
		}
	}
	return g
}

func leaves(n *vdf.Node) map[string]string {
	m := map[string]string{}
	if n == nil {
		return m
	}
	for _, c := range n.Children {
		if !c.IsBlock() {
			m[c.Key] = c.Value
		}
	}
	return m
}

// Base returns the first action set (preset id 0 in practice).
func (c *Config) Base() *Set {
	for _, s := range c.Sets {
		if !s.Layer {
			return s
		}
	}
	return nil
}

// SetByTarget resolves the 1-based index used by CHANGE_PRESET / *_layer bindings.
func (c *Config) SetByTarget(n int) *Set {
	for _, s := range c.Sets {
		if s.ID == n-1 {
			return s
		}
	}
	return nil
}

// groupFor returns the active group bound to source in set s.
func (c *Config) groupFor(s *Set, modeShift bool, source string) *Group {
	var normal, shifted *Group
	for _, sb := range s.Sources {
		if !sb.Active || sb.Source != source {
			continue
		}
		if sb.ModeShift {
			shifted = c.Groups[sb.GroupID]
		} else {
			normal = c.Groups[sb.GroupID]
		}
	}
	if modeShift && shifted != nil {
		return shifted
	}
	return normal
}

// chain lists the groups that apply to source, topmost layer first, ending at
// the group that defines the source's mode. A layer group with no inputs and
// the same mode as the group beneath it is transparent.
func (c *Config) chain(base *Set, layers []*Set, modeShift bool, source string) []*Group {
	var stack []*Group
	if g := c.groupFor(base, modeShift, source); g != nil {
		stack = append(stack, g)
	}
	for _, l := range layers {
		g := c.groupFor(l, modeShift, source)
		if g == nil {
			continue
		}
		if len(stack) > 0 && g.Mode != stack[len(stack)-1].Mode {
			stack = stack[:0] // a different mode replaces the source outright
		}
		stack = append(stack, g)
	}
	slices.Reverse(stack)
	return stack
}

// ActiveGroup returns the group that currently defines source's mode.
func (c *Config) ActiveGroup(base *Set, layers []*Set, modeShift bool, source string) *Group {
	ch := c.chain(base, layers, modeShift, source)
	for _, g := range ch {
		if len(g.Inputs) > 0 {
			return g
		}
	}
	if len(ch) > 0 {
		return ch[0]
	}
	return nil
}

// Activators returns what input on source does, letting unset inputs in
// layer groups fall through to the groups below.
func (c *Config) Activators(base *Set, layers []*Set, modeShift bool, source, input string) []Activator {
	for _, g := range c.chain(base, layers, modeShift, source) {
		if acts, ok := g.Inputs[input]; ok {
			return acts
		}
	}
	return nil
}

// MenuItem is one entry of a radial or touch menu.
type MenuItem struct {
	Index      int // N from touch_menu_button_N
	Input      string
	Label      string
	Icon       string
	Bindings   []binding.Binding // bindings of the first activator
	Activators []Activator
}

// MenuItems returns a menu group's items ordered by index.
func (g *Group) MenuItems() []MenuItem {
	var items []MenuItem
	for name, acts := range g.Inputs {
		n, ok := strings.CutPrefix(name, "touch_menu_button_")
		if !ok {
			continue
		}
		idx, err := strconv.Atoi(n)
		if err != nil {
			continue
		}
		it := MenuItem{Index: idx, Input: name, Activators: acts, Bindings: acts[0].Bindings,
			Label: binding.Label(acts[0].Bindings)}
		for _, b := range it.Bindings {
			if it.Icon == "" {
				it.Icon = b.Icon
			}
		}
		items = append(items, it)
	}
	slices.SortFunc(items, func(a, b MenuItem) int { return a.Index - b.Index })
	return items
}

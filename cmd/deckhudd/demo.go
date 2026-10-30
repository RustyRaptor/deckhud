package main

import (
	"context"
	"time"

	"github.com/RustyRaptor/deckhud/internal/engine"
	"github.com/RustyRaptor/deckhud/internal/hid"
)

// demo drives the real engine with a scripted controller session so the
// overlay can be previewed without touching the Deck.
func demo(ctx context.Context, uiDir string) error {
	opts := runOpts{uiDir: uiDir, print: true, script: demoScript}
	return run(ctx, opts)
}

type demoStep struct {
	wait  time.Duration
	apply func(*hid.State)
}

var demoScript = []demoStep{
	{2500 * time.Millisecond, func(s *hid.State) { s.RightStick = hid.Vec{X: 0, Y: 30000} }},
	{700 * time.Millisecond, func(s *hid.State) { s.RightStick = hid.Vec{X: 26000, Y: 18000} }},
	{700 * time.Millisecond, func(s *hid.State) { s.RightStick = hid.Vec{X: 30000, Y: -2000} }},
	{900 * time.Millisecond, func(s *hid.State) { s.RightStick = hid.Vec{} }},
	{1200 * time.Millisecond, func(s *hid.State) { s.Buttons |= hid.R4 }},
	{100 * time.Millisecond, func(s *hid.State) { s.Buttons &^= hid.R4 }},
	{600 * time.Millisecond, func(s *hid.State) { s.Buttons |= hid.A }},
	{100 * time.Millisecond, func(s *hid.State) { s.Buttons &^= hid.A }},
	{1500 * time.Millisecond, func(s *hid.State) { s.Buttons |= hid.L4 }},
	{800 * time.Millisecond, func(s *hid.State) { s.Buttons &^= hid.L4 }},
	{2500 * time.Millisecond, nil},
}

// playScript feeds scripted states into the engine at device rate.
func (d *daemon) playScript(ctx context.Context, script []demoStep) {
	var s hid.State
	tick := time.NewTicker(4 * time.Millisecond)
	defer tick.Stop()
	for _, step := range script {
		end := time.Now().Add(step.wait)
		for time.Now().Before(end) {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
			d.feed(s)
		}
		if step.apply != nil {
			step.apply(&s)
		}
	}
}

func (d *daemon) feed(s hid.State) {
	d.mu.Lock()
	var evs []engine.Event
	if d.eng != nil {
		evs = d.eng.Update(s, time.Now())
	}
	d.mu.Unlock()
	for _, e := range evs {
		d.emit(e)
	}
}

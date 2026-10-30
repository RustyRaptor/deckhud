// Package appwatch follows Steam's controller log to learn which app's
// controller config is active and which configset files apply.
package appwatch

import (
	"path/filepath"
	"slices"
	"strings"
)

const (
	activationMarker = "Queueing activation for controller: "
	configSetMarker  = "ConfigSet - found config set file on-disk: "
)

// State is what the log has told us so far.
type State struct {
	App        string   // Steam app id whose controller config is active
	ConfigSets []string // configset file names, highest precedence first

	pending []string // configset lines of a burst still being read
	inBurst bool
}

// Feed consumes one log line and reports whether App or ConfigSets changed.
// Steam logs configsets as a burst of consecutive lines; the new list takes
// effect on the first line after the burst.
func (s *State) Feed(line string) bool {
	if _, path, ok := strings.Cut(line, configSetMarker); ok {
		if !s.inBurst {
			s.pending = nil
			s.inBurst = true
		}
		s.pending = append(s.pending, filepath.Base(strings.TrimSpace(path)))
		return false
	}
	changed := s.Flush()
	if _, rest, ok := strings.Cut(line, activationMarker); ok {
		if _, app, ok := strings.Cut(rest, "app: "); ok {
			if app = strings.TrimSpace(app); app != "" && app != s.App {
				s.App = app
				changed = true
			}
		}
	}
	return changed
}

// Flush ends a pending configset burst, e.g. when the reader hits EOF.
func (s *State) Flush() bool {
	if !s.inBurst {
		return false
	}
	s.inBurst = false
	if slices.Equal(s.pending, s.ConfigSets) {
		return false
	}
	s.ConfigSets = s.pending
	return true
}

package appwatch

import (
	"reflect"
	"strings"
	"testing"
)

const sample = `[2026-10-30 14:43:54] Add to Config Cache Request 15 413080
[2026-10-30 14:43:54] Queueing activation for controller: 15 app: 413080
[2026-10-30 14:44:00] ConfigSet - failed to find config set file on-disk: /x/config/configset_OLD.vdf
[2026-10-30 14:44:00] Set Account Config Sets 15 0 0
[2026-10-30 14:44:00] ConfigSet - found config set file on-disk: /x/config/configset_FY1S3420A2CF.vdf
[2026-10-30 14:44:00] ConfigSet - found config set file on-disk: /x/config/configset_controller_neptune.vdf
[2026-10-30 14:44:00] Set Account Config Sets 15 0 0
[2026-10-30 15:30:42] Queueing activation for controller: 15 app: 769
[2026-10-30 15:31:17] Queueing activation for controller: 15 app: 413080
`

func TestFeed(t *testing.T) {
	var s State
	changed := 0
	for _, line := range strings.Split(sample, "\n") {
		if s.Feed(line) {
			changed++
		}
	}
	if s.App != "413080" {
		t.Errorf("App = %q", s.App)
	}
	want := []string{"configset_FY1S3420A2CF.vdf", "configset_controller_neptune.vdf"}
	if !reflect.DeepEqual(s.ConfigSets, want) {
		t.Errorf("ConfigSets = %v", s.ConfigSets)
	}
	// app 413080, configsets, app 769, app 413080
	if changed != 4 {
		t.Errorf("changed %d times, want 4", changed)
	}
}

func TestFeedNewBurstReplacesConfigSets(t *testing.T) {
	var s State
	s.Feed("[t] ConfigSet - found config set file on-disk: /a/configset_A.vdf")
	s.Feed("[t] ConfigSet - found config set file on-disk: /a/configset_controller_neptune.vdf")
	s.Feed("[t] something else")
	s.Feed("[t] ConfigSet - found config set file on-disk: /a/configset_controller_neptune.vdf")
	if !s.Flush() {
		t.Error("Flush should report the changed burst")
	}
	if !reflect.DeepEqual(s.ConfigSets, []string{"configset_controller_neptune.vdf"}) {
		t.Errorf("ConfigSets = %v", s.ConfigSets)
	}
}

func TestFeedRepeatedAppIsNotAChange(t *testing.T) {
	var s State
	s.Feed("[t] Queueing activation for controller: 15 app: 413080")
	if s.Feed("[t] Queueing activation for controller: 15 app: 413080") {
		t.Error("same app again should not report a change")
	}
}

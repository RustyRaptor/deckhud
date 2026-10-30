package vdf

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseLocalSteamConfigs parses every .vdf under $DECKHUD_SCAN_DIR. It is a
// smoke test against real-world files and is skipped unless the variable is set.
func TestParseLocalSteamConfigs(t *testing.T) {
	dir := os.Getenv("DECKHUD_SCAN_DIR")
	if dir == "" {
		t.Skip("set DECKHUD_SCAN_DIR to scan local Steam configs")
	}
	n := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".vdf" {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err := Parse(f); err != nil {
			t.Errorf("%s: %v", path, err)
		}
		n++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("parsed %d files", n)
}

func TestParseFixtures(t *testing.T) {
	files, _ := filepath.Glob("../../testdata/vdf/*.vdf")
	if len(files) == 0 {
		t.Fatal("no fixtures found")
	}
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Parse(f); err != nil {
			t.Errorf("%s: %v", path, err)
		}
		f.Close()
	}
}

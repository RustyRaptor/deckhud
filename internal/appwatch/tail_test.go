package appwatch

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func appendFile(t *testing.T, path, s string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(s); err != nil {
		t.Fatal(err)
	}
}

func TestTailReturnsOnlyNewCompleteLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "controller.txt")
	appendFile(t, path, "a\nb\n")
	tl := &Tail{Path: path}

	lines, err := tl.Poll()
	if err != nil || !reflect.DeepEqual(lines, []string{"a", "b"}) {
		t.Fatalf("first poll = %q, %v", lines, err)
	}
	if lines, _ := tl.Poll(); len(lines) != 0 {
		t.Errorf("no growth should yield nothing, got %q", lines)
	}

	appendFile(t, path, "c\npart")
	if lines, _ := tl.Poll(); !reflect.DeepEqual(lines, []string{"c"}) {
		t.Errorf("partial line must wait, got %q", lines)
	}
	appendFile(t, path, "ial\n")
	if lines, _ := tl.Poll(); !reflect.DeepEqual(lines, []string{"partial"}) {
		t.Errorf("got %q", lines)
	}
}

func TestTailRestartsAfterTruncationOrRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "controller.txt")
	appendFile(t, path, "old line one\nold line two\n")
	tl := &Tail{Path: path}
	tl.Poll()

	if err := os.WriteFile(path, []byte("new\n"), 0o644); err != nil { // Steam restarted
		t.Fatal(err)
	}
	if lines, _ := tl.Poll(); !reflect.DeepEqual(lines, []string{"new"}) {
		t.Errorf("got %q", lines)
	}
}

func TestTailMissingFile(t *testing.T) {
	tl := &Tail{Path: filepath.Join(t.TempDir(), "nope.txt")}
	if _, err := tl.Poll(); err == nil {
		t.Error("expected error for missing file")
	}
}

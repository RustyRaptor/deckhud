package hid

import (
	"os"
	"path/filepath"
	"testing"
)

func fakeHidraw(t *testing.T, root, name, hidID string, desc []byte) {
	t.Helper()
	dir := filepath.Join(root, "class", "hidraw", name, "device")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	uevent := "DRIVER=hid-steam\nHID_ID=" + hidID + "\nHID_NAME=Valve Software Steam Controller\n"
	if err := os.WriteFile(filepath.Join(dir, "uevent"), []byte(uevent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "report_descriptor"), desc, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFindPicksVendorInterfaceOfDeck(t *testing.T) {
	sys := t.TempDir()
	deck := "0003:000028DE:00001205"
	fakeHidraw(t, sys, "hidraw0", deck, []byte{0x05, 0x01, 0x09, 0x02}) // lizard mouse
	fakeHidraw(t, sys, "hidraw1", deck, []byte{0x05, 0x01, 0x09, 0x06}) // lizard keyboard
	fakeHidraw(t, sys, "hidraw3", "0003:0000046D:0000C077", []byte{0x06, 0xff, 0xff, 0x09})
	fakeHidraw(t, sys, "hidraw2", deck, []byte{0x06, 0xff, 0xff, 0x09}) // controller

	got, err := Find(sys)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/dev/hidraw2" {
		t.Errorf("Find = %q, want /dev/hidraw2", got)
	}
}

func TestFindNoDeck(t *testing.T) {
	sys := t.TempDir()
	fakeHidraw(t, sys, "hidraw0", "0003:0000046D:0000C077", []byte{0x05, 0x01})
	if _, err := Find(sys); err == nil {
		t.Fatal("expected error when no Deck controller is present")
	}
}

func TestStreamSkipsNonInputAndStopsAtEOF(t *testing.T) {
	a, b, status := blank(), blank(), blank()
	a[8] = 0x80 // A
	status[2] = 0x04
	b[13] = 0x02 // L4
	var buf []byte
	for _, r := range [][]byte{a, status, b} {
		buf = append(buf, r...)
	}
	var got []Button
	err := Stream(bytesReader(buf), func(s State) { got = append(got, s.Buttons) })
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if len(got) != 2 || got[0] != A || got[1] != L4 {
		t.Errorf("got %v", got)
	}
}

package hid

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Valve's USB IDs for the Steam Deck's built-in controller.
const deckHIDID = "000028DE:00001205"

// vendorUsagePage is the report-descriptor prefix of the interface that
// carries controller state (Usage Page 0xFFFF). The other two interfaces are
// the "lizard mode" mouse and keyboard.
var vendorUsagePage = []byte{0x06, 0xff, 0xff}

// Find returns the /dev path of the Deck controller's state interface.
// sysRoot is normally "/sys".
func Find(sysRoot string) (string, error) {
	dirs, err := filepath.Glob(filepath.Join(sysRoot, "class", "hidraw", "hidraw*"))
	if err != nil {
		return "", err
	}
	for _, d := range dirs {
		uevent, err := os.ReadFile(filepath.Join(d, "device", "uevent"))
		if err != nil || !strings.Contains(strings.ToUpper(string(uevent)), "HID_ID=0003:"+deckHIDID) {
			continue
		}
		desc, err := os.ReadFile(filepath.Join(d, "device", "report_descriptor"))
		if err != nil || !bytes.HasPrefix(desc, vendorUsagePage) {
			continue
		}
		return "/dev/" + filepath.Base(d), nil
	}
	return "", fmt.Errorf("hid: no Steam Deck controller found under %s", sysRoot)
}

// Stream reads reports from r until EOF or error, calling f for every decoded
// controller state. Non-state reports are skipped. A clean EOF returns nil.
func Stream(r io.Reader, f func(State)) error {
	buf := make([]byte, ReportSize)
	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		s, err := Decode(buf)
		if errors.Is(err, ErrNotInput) {
			continue
		}
		if err != nil {
			return err
		}
		f(s)
	}
}

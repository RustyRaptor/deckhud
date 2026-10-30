package appwatch

import (
	"bytes"
	"io"
	"os"
)

// Tail reads lines appended to a file between polls. If the file shrinks
// (Steam restarted and rewrote its log) it starts again from the top.
type Tail struct {
	Path string

	off     int64
	partial []byte
}

// Poll returns the complete lines added since the last call.
func (t *Tail) Poll() ([]string, error) {
	f, err := os.Open(t.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if st.Size() < t.off {
		t.off, t.partial = 0, nil
	}
	if st.Size() == t.off {
		return nil, nil
	}
	data, err := io.ReadAll(io.NewSectionReader(f, t.off, st.Size()-t.off))
	if err != nil {
		return nil, err
	}
	t.off += int64(len(data))
	data = append(t.partial, data...)

	var lines []string
	for {
		i := bytes.IndexByte(data, '\n')
		if i < 0 {
			break
		}
		lines = append(lines, string(bytes.TrimRight(data[:i], "\r")))
		data = data[i+1:]
	}
	t.partial = append([]byte(nil), data...)
	return lines, nil
}

package nativemessaging

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"testing"
)

func frame(body string) []byte {
	return append(binary.NativeEndian.AppendUint32(nil, uint32(len(body))), body...) //nolint:gosec // G115: test bodies are short
}

func TestWriteThenReadRoundTrips(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	in := map[string]string{"link": "nxm://x/mods/1"}
	for range 2 {
		if err := Write(&buf, in); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.HasPrefix(buf.Bytes(), frame(`{"link":"nxm://x/mods/1"}`)) {
		t.Fatalf("frame % x", buf.Bytes())
	}
	for range 2 {
		var out map[string]string
		if err := Read(&buf, 64, &out); err != nil || out["link"] != in["link"] {
			t.Fatalf("read %v, %v", out, err)
		}
	}
	var out any
	if err := Read(&buf, 64, &out); !errors.Is(err, io.EOF) {
		t.Fatalf("clean end: %v", err)
	}
}

func TestReadRefusesBadFrames(t *testing.T) {
	t.Parallel()
	for name, in := range map[string][]byte{
		"over the limit":   frame(`"` + string(make([]byte, 20)) + `"`),
		"short prefix":     {1, 0},
		"short body":       frame(`{"a":1}`)[:6],
		"not json":         frame(`{`),
		"trailing in body": frame(`{} {}`),
		"huge length":      {0xff, 0xff, 0xff, 0xff},
	} {
		var v any
		err := Read(bytes.NewReader(in), 16, &v)
		if err == nil || errors.Is(err, io.EOF) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// FuzzRead holds that no byte stream from a browser makes Read panic or accept a message over its limit.
func FuzzRead(f *testing.F) {
	f.Add(frame(`{"link":"nxm://stardewvalley/mods/1/files/2"}`))
	f.Add(slices.Concat(frame(`{"type":"mod","modId":1}`), frame(`[1,2]`)))
	f.Add([]byte{0xff, 0xff, 0xff, 0x7f})
	f.Add(frame(`"\u0000"`))
	const limit = 1024
	f.Fuzz(func(t *testing.T, in []byte) {
		r := bytes.NewReader(in)
		for {
			before := r.Len()
			var v json.RawMessage
			if err := Read(r, limit, &v); err != nil {
				return
			}
			if before-r.Len() > 4+limit {
				t.Fatalf("read %d bytes for one message", before-r.Len())
			}
		}
	})
}

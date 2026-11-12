package nativemessaging

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"
)

func FuzzWriteRead(f *testing.F) {
	f.Add([]byte(`{"type":"ping"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`"hello"`))
	f.Add([]byte(`null`))
	f.Add([]byte(`true`))
	f.Add([]byte(`0`))
	f.Fuzz(func(t *testing.T, payload []byte) {
		if len(payload) > 1<<16 {
			return
		}
		var v any
		if err := json.Unmarshal(payload, &v); err != nil {
			return
		}
		want, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := Write(&buf, v); err != nil {
			t.Fatal(err)
		}
		if buf.Len() != 4+len(want) {
			t.Fatalf("Write frame %d bytes, want 4+%d", buf.Len(), len(want))
		}
		var got any
		if err := Read(&buf, buf.Len(), &got); err != nil {
			t.Fatalf("Read after Write: %v", err)
		}
		have, err := json.Marshal(got)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(want, have) {
			t.Fatalf("round-trip JSON: got %s want %s", have, want)
		}
		if err := Read(&buf, 1<<20, &got); !errors.Is(err, io.EOF) {
			t.Fatalf("second Read: got %v want EOF", err)
		}
	})
}

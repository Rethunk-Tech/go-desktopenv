package nativemessaging

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
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

func FuzzStateOf(f *testing.F) {
	f.Add("/opt/app", []byte(`{"path":"/opt/app"}`), false)
	f.Add("/opt/app", []byte(`{"path":"/other"}`), false)
	f.Add("/opt/app", []byte(`{}`), false)
	f.Add("/opt/app", []byte(`not json`), false)
	f.Add("/opt/app", []byte(`{"path":""}`), false)
	f.Add("/opt/app", []byte{}, true)
	f.Fuzz(func(t *testing.T, exe string, b []byte, missing bool) {
		var err error
		if missing {
			err = fs.ErrNotExist
		}
		s := stateOf(exe, b, err)
		switch s {
		case OK, Missing, Stale, Unreadable:
		default:
			t.Fatalf("unknown state %q", s)
		}
		if missing && s != Missing {
			t.Fatalf("ErrNotExist gave %q", s)
		}
		if !missing && s == Missing {
			t.Fatalf("nil read error gave Missing")
		}
	})
}

func FuzzCheckName(f *testing.F) {
	for _, n := range []string{"com.example.host", "a_b.c1", "Com.Example", "my-host", "host.", ".host", "a..b", "", "org.example.app"} {
		f.Add(n)
	}
	f.Fuzz(func(t *testing.T, name string) {
		err := (Host{Name: name}).checkName()
		if err == nil {
			if name != filepath.Base(name) || strings.Contains(name, "..") || name[0] == '.' || name[len(name)-1] == '.' {
				t.Fatalf("checkName accepted %q", name)
			}
			return
		}
		if !errors.Is(err, ErrBadName) {
			t.Fatalf("checkName(%q) = %v, want ErrBadName", name, err)
		}
	})
}

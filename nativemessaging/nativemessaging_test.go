package nativemessaging

import (
	"errors"
	"runtime"
	"testing"
)

func TestCheckName(t *testing.T) {
	tests := []struct {
		name string
		ok   bool
	}{
		{"com.example.host", true},
		{"a_b.c1", true},
		{"z0.9_host", true},
		{"Com.Example", false},
		{"my-host", false},
		{"host.", false},
		{".host", false},
		{"a..b", false},
		{"", false},
	}
	for _, tt := range tests {
		err := (Host{Name: tt.name}).checkName()
		if tt.ok {
			if err != nil {
				t.Errorf("checkName(%q) = %v, want nil", tt.name, err)
			}
			continue
		}
		if !errors.Is(err, ErrBadName) {
			t.Errorf("checkName(%q) = %v, want ErrBadName", tt.name, err)
		}
	}
}

func TestStateOfTreatsACaseFoldedPathAsStaleOffWindows(t *testing.T) {
	t.Parallel()
	want := Stale
	if runtime.GOOS == "windows" {
		want = OK
	}
	if got := stateOf("/app", []byte(`{"path":"/App"}`), nil); got != want {
		t.Errorf("stateOf = %q, want %q", got, want)
	}
}

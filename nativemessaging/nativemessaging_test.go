package nativemessaging

import (
	"errors"
	"testing"
)

func TestCheckName(t *testing.T) {
	tests := []struct {
		name string
		ok   bool
	}{
		{"com.example.host", true},
		{"a_b.c1", true},
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

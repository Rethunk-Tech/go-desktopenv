package firewall

import (
	"errors"
	"runtime"
	"testing"
)

func TestOnlyPublic(t *testing.T) {
	tests := []struct {
		current int32
		want    bool
	}{
		{ProfilePublic, true},
		{ProfileDomain, false},
		{ProfilePrivate, false},
		{ProfilePublic | ProfileDomain, false},
		{ProfilePublic | ProfilePrivate, false},
		{ProfilesLocal, false},
		{ProfilesEvery, false},
		{ProfilesAll, false},
		{0, false},
	}
	for _, tt := range tests {
		if got := OnlyPublic(tt.current); got != tt.want {
			t.Fatalf("OnlyPublic(%d) = %v, want %v", tt.current, got, tt.want)
		}
	}
}

func TestBlocked(t *testing.T) {
	const exe = `C:\App\app.exe`
	allowLocal := Rule{App: exe, Inbound: true, Allow: true, Enabled: true, Profiles: ProfilesLocal}
	allowAll := Rule{App: exe, Inbound: true, Allow: true, Enabled: true, Profiles: ProfilesAll}
	allowEvery := Rule{App: exe, Inbound: true, Allow: true, Enabled: true, Profiles: ProfilesEvery}
	allowDomain := Rule{App: exe, Inbound: true, Allow: true, Enabled: true, Profiles: ProfileDomain}
	blockPublic := Rule{App: exe, Inbound: true, Allow: false, Enabled: true, Profiles: ProfilePublic}
	blockLocal := Rule{App: exe, Inbound: true, Allow: false, Enabled: true, Profiles: ProfilesLocal}
	blockAll := Rule{App: exe, Inbound: true, Allow: false, Enabled: true, Profiles: ProfilesAll}
	disabled := Rule{App: exe, Inbound: true, Allow: true, Enabled: false, Profiles: ProfilesLocal}
	outbound := Rule{App: exe, Inbound: false, Allow: true, Enabled: true, Profiles: ProfilesLocal}
	other := Rule{App: `C:\other.exe`, Inbound: true, Allow: true, Enabled: true, Profiles: ProfilesLocal}
	folded := Rule{App: `C:\APP\APP.EXE`, Inbound: true, Allow: true, Enabled: true, Profiles: ProfilesLocal}

	tests := []struct {
		name    string
		rules   []Rule
		anyAddr bool
		want    bool
	}{
		{"no rules", nil, false, true},
		{"no rules anyAddr", nil, true, true},
		{"lan ready", []Rule{allowLocal, blockPublic}, false, false},
		{"lan missing public block", []Rule{allowLocal}, false, true},
		{"anyAddr profilesAll", []Rule{allowAll}, true, false},
		{"anyAddr profilesEvery", []Rule{allowEvery}, true, false},
		{"anyAddr local only", []Rule{allowLocal, blockPublic}, true, true},
		{"block overlaps reach", []Rule{allowLocal, blockPublic, blockLocal}, false, true},
		{"block all overlaps", []Rule{allowLocal, blockAll}, false, true},
		{"disabled allow", []Rule{disabled, blockPublic}, false, true},
		{"outbound ignored", []Rule{outbound, blockPublic}, false, true},
		{"other exe ignored", []Rule{other, blockPublic}, false, true},
		{"case fold exe", []Rule{folded, blockPublic}, false, false},
		{"allow domain only", []Rule{allowDomain, blockPublic}, false, true},
		{"profilesAll allow lan", []Rule{allowAll, blockPublic}, false, false},
		{"anyAddr public block overlaps", []Rule{allowAll, blockPublic}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Blocked(tt.rules, exe, tt.anyAddr); got != tt.want {
				t.Fatalf("Blocked(...) = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddRejectsEmptyExeAndNames(t *testing.T) {
	for _, c := range []struct{ exe, name, pub string }{
		{"", "n", "p"},
		{`C:\a.exe`, "", "p"},
		{`C:\a.exe`, "n", ""},
	} {
		err := Add(c.exe, c.name, c.pub, false)
		if err == nil || errors.Is(err, ErrUnsupported) {
			t.Errorf("Add(%q, %q, %q) = %v, want a validation error", c.exe, c.name, c.pub, err)
		}
	}
}

func TestUnsupported(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stubs are for other operating systems")
	}
	if _, err := CurrentProfileTypes(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("CurrentProfileTypes() error = %v, want ErrUnsupported", err)
	}
	if _, err := Rules(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Rules() error = %v, want ErrUnsupported", err)
	}
	if err := Add("exe", "name", "public", false); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Add() error = %v, want ErrUnsupported", err)
	}
	if err := Elevate("exe", "args"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Elevate() error = %v, want ErrUnsupported", err)
	}
}

package flatpak

import (
	"context"
	"reflect"
	"testing"
)

func TestHostArgv(t *testing.T) {
	t.Parallel()
	name, args := HostArgv("/g", []string{"A=1"}, "steam", "-applaunch", "413150")
	want := []string{"--host", "--directory=/g", "--env=A=1", "steam", "-applaunch", "413150"}
	if name != "flatpak-spawn" || !reflect.DeepEqual(args, want) {
		t.Fatalf("%s %v", name, args)
	}
	_, args = HostArgv("", nil, "which", "steam")
	if !reflect.DeepEqual(args, []string{"--host", "which", "steam"}) {
		t.Fatalf("%v", args)
	}
}

func TestInside(t *testing.T) {
	t.Setenv("FLATPAK_ID", "")
	if Inside() {
		t.Fatal("no FLATPAK_ID")
	}
	t.Setenv("FLATPAK_ID", "org.example.App")
	if !Inside() {
		t.Fatal("FLATPAK_ID set")
	}
}

func TestCommandContextHostTerm(t *testing.T) {
	t.Parallel()
	name, args := HostArgv("", nil, "sleep", "1")
	cmd := CommandContext(context.Background(), name, args...)
	if cmd.Cancel == nil || cmd.WaitDelay == 0 {
		t.Fatalf("host command lacks Cancel/WaitDelay: %v %v", cmd.Cancel == nil, cmd.WaitDelay)
	}
	if plain := CommandContext(context.Background(), "sleep", "1"); plain.WaitDelay != 0 {
		t.Fatal("non-host command should keep the default kill")
	}
}

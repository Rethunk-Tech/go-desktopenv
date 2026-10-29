package nowindow

import (
	"os/exec"
	"syscall"
	"testing"

	"golang.org/x/sys/windows"
)

func TestSetKeepsExistingAttributesAndAddsTheFlag(t *testing.T) {
	t.Parallel()
	cmd := exec.CommandContext(t.Context(), "cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	Set(cmd)
	if !cmd.SysProcAttr.HideWindow || cmd.SysProcAttr.CreationFlags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatalf("%+v", cmd.SysProcAttr)
	}
}

func TestSetAllocatesAttributes(t *testing.T) {
	t.Parallel()
	cmd := exec.CommandContext(t.Context(), "cmd")
	Set(cmd)
	if cmd.SysProcAttr == nil || cmd.SysProcAttr.CreationFlags != windows.CREATE_NO_WINDOW {
		t.Fatalf("%+v", cmd.SysProcAttr)
	}
}

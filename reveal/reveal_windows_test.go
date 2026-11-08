package reveal

import (
	"os/exec"
	"testing"
)

func TestSetArgsQuotesAPathWithAComma(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), "explorer")
	dir := `C:\Users\me\My,Files`
	setArgs(cmd, dir)
	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr is nil")
	}
	want := `explorer "` + dir + `"`
	if cmd.SysProcAttr.CmdLine != want {
		t.Errorf("CmdLine = %q, want %q", cmd.SysProcAttr.CmdLine, want)
	}
}

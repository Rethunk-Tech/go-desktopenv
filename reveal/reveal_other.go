//go:build !windows

package reveal

import "os/exec"

func setArgs(*exec.Cmd, string) {}

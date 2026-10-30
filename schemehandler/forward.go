package schemehandler

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/go-desktopenv/desktopentry"
)

// windowsForwardArgv is argv for the saved Windows handler, with link as one element in place of %1.
func windowsForwardArgv(previousSaved, link string) (string, []string, error) {
	p := splitPrevious(previousSaved)
	if strings.TrimSpace(p.cmd) == "" {
		return "", nil, errNoPrevious
	}
	argv := commandLineToArgv(p.cmd)
	if len(argv) == 0 || argv[0] == "" {
		return "", nil, errNoPrevious
	}
	return programAndArgs(argv, "%1", link)
}

// linuxForwardArgv is argv for the saved desktop Exec, with link as one element in place of %u or %U.
func linuxForwardArgv(previousDesktopID, link string, execLine func(string) (string, error)) (string, []string, error) {
	if previousDesktopID == "" {
		return "", nil, errNoPrevious
	}
	exec, err := execLine(previousDesktopID)
	if err != nil {
		return "", nil, err
	}
	argv := desktopentry.SplitExec(exec)
	if len(argv) == 0 || argv[0] == "" {
		return "", nil, fmt.Errorf("desktop entry %q has no Exec", previousDesktopID)
	}
	return programAndArgs(argv, "", link)
}

func programAndArgs(argv []string, winField, link string) (string, []string, error) {
	placed := false
	out := make([]string, 0, len(argv)+1)
	for _, a := range argv {
		if winField != "" {
			if before, after, ok := strings.Cut(a, winField); ok {
				out = append(out, before+link+after)
				placed = true
				continue
			}
			out = append(out, a)
			continue
		}
		switch a {
		case "%u", "%U":
			out = append(out, link)
			placed = true
		case "%f", "%F", "%i", "%c", "%k":
		default:
			out = append(out, a)
		}
	}
	if !placed {
		out = append(out, link)
	}
	return out[0], out[1:], nil
}

// commandLineToArgv splits a Windows command line under CommandLineToArgvW rules.
func commandLineToArgv(cmd string) []string {
	var args []string
	var arg strings.Builder
	inQuotes := false
	bs := 0
	started := false
	flush := func() {
		args = append(args, arg.String())
		arg.Reset()
		started = false
	}
	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		if c == '\\' {
			bs++
			started = true
			continue
		}
		if c == '"' {
			arg.WriteString(strings.Repeat(`\`, bs/2))
			if bs%2 == 0 {
				inQuotes = !inQuotes
			} else {
				arg.WriteByte('"')
			}
			bs = 0
			started = true
			continue
		}
		if bs > 0 {
			arg.WriteString(strings.Repeat(`\`, bs))
			bs = 0
		}
		if (c == ' ' || c == '\t') && !inQuotes {
			if started {
				flush()
			}
			continue
		}
		arg.WriteByte(c)
		started = true
	}
	if bs > 0 {
		arg.WriteString(strings.Repeat(`\`, bs))
		started = true
	}
	if started {
		flush()
	}
	return args
}

// previous is what a scheme's key held before this program took it over. Saved as "command\nicon\nname"; a value
// saved with fewer lines restores only the parts it holds.
type previous struct {
	cmd, icon, name  string
	hasIcon, hasName bool
}

func previousID(cmd, icon, name string) string {
	if cmd == "" {
		return ""
	}
	return cmd + "\n" + icon + "\n" + name
}

func splitPrevious(saved string) previous {
	if saved == "" {
		return previous{}
	}
	parts := strings.SplitN(saved, "\n", 3)
	p := previous{cmd: parts[0], hasIcon: len(parts) > 1, hasName: len(parts) > 2}
	if p.hasIcon {
		p.icon = parts[1]
	}
	if p.hasName {
		p.name = parts[2]
	}
	return p
}

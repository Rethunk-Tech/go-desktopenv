package schemehandler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/go-desktopenv/desktopentry"
)

// windowsForwardArgv is argv for the saved Windows handler, with link as one element in place of %1.
func windowsForwardArgv(previousSaved, link string) (string, []string, error) {
	if err := checkLink(link); err != nil {
		return "", nil, err
	}
	p := splitPrevious(previousSaved)
	if strings.TrimSpace(p.cmd) == "" {
		return "", nil, errNoPrevious
	}
	argv := commandLineToArgv(p.cmd)
	if len(argv) == 0 || argv[0] == "" {
		return "", nil, errNoPrevious
	}
	if runsThroughCmd(argv[0]) {
		var err error
		if link, err = escapeForCmd(link); err != nil {
			return "", nil, err
		}
	}
	return programAndArgs(argv, "%1", link)
}

// escapeForCmd makes link one literal word for cmd.exe, which parses the command line again after Go has escaped it for
// the C runtime. A caret before each of & | < > ( ) ^ ! keeps cmd from reading them as operators, so the '&' in a real
// nxm link survives. What no escape can neutralise is refused: a quote, a line break, white space (Go would quote the
// word, where carets turn literal), and '%' before a letter or underscore, which cmd expands as an environment variable.
func escapeForCmd(link string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(link); i++ {
		c := link[i]
		switch {
		case c == '"' || c == '\r' || c == '\n' || c == ' ' || c == '\t' || c < 0x20:
			return "", fmt.Errorf("schemehandler: the link holds %q, which cmd.exe cannot be given as one word", c)
		case c == '%' && i+1 < len(link) && (link[i+1] == '_' || link[i+1] >= 'A' && link[i+1] <= 'Z' || link[i+1] >= 'a' && link[i+1] <= 'z'):
			return "", errors.New("schemehandler: the link holds a % that cmd.exe would expand as a variable")
		case strings.IndexByte("&|<>()^!", c) >= 0:
			b.WriteByte('^')
		}
		b.WriteByte(c)
	}
	return b.String(), nil
}

// runsThroughCmd reports whether program is cmd.exe or a batch file, which re-parse their arguments as a command line
// after Go has escaped them for the C runtime.
func runsThroughCmd(program string) bool {
	base := strings.ToLower(program[strings.LastIndexAny(program, `\/`)+1:])
	return base == "cmd" || base == "cmd.exe" || strings.HasSuffix(base, ".cmd") || strings.HasSuffix(base, ".bat")
}

// linuxForwardArgv is argv for the saved desktop Exec, with link as one element in place of %u or %U.
func linuxForwardArgv(previousDesktopID, link string, execLine func(string) (string, error)) (string, []string, error) {
	if previousDesktopID == "" {
		return "", nil, errNoPrevious
	}
	if err := checkLink(link); err != nil {
		return "", nil, err
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

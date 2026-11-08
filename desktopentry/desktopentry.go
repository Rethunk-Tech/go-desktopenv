// Package desktopentry escapes and reads the string values of a freedesktop Desktop Entry file, writes one from its
// keys, and rewrites a single key while keeping the rest.
package desktopentry

import (
	"errors"
	"strings"
)

var (
	// valueEscaper doubles a backslash, which starts an escape in the file format, and turns a line break into a
	// space, since one would end the key.
	valueEscaper = strings.NewReplacer(`\`, `\\`, "\t", `\t`, "\n", " ", "\r", " ")
	// execLineEscaper keeps a line break in an Exec path as itself, not as the space Value writes, so the program
	// path is not changed.
	execLineEscaper = strings.NewReplacer(`\`, `\\`, "\t", `\t`, "\n", `\n`, "\r", `\r`)
	// quotedEscaper escapes what the Exec quoting treats as special inside double quotes, and doubles a percent so a
	// literal one is not read as a field code.
	quotedEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`, `%`, `%%`)
	// stringUnescaper undoes the string-value escapes. An unknown escape is left as written.
	stringUnescaper = strings.NewReplacer(`\\`, `\`, `\s`, " ", `\n`, "\n", `\t`, "\t", `\r`, "\r")
)

// execReserved are the characters that force an Exec argument into double quotes.
const execReserved = " \t\n\r\"'\\><~|&;$*?#()`"

// Value escapes s as a Desktop Entry string value: `\` doubled, a tab as `\t`, a leading space as `\s`, and a line
// break (CR or LF) as a space, since one would end the key.
func Value(s string) string {
	s = valueEscaper.Replace(s)
	if rest, ok := strings.CutPrefix(s, " "); ok {
		return `\s` + rest
	}
	return s
}

// Exec builds an Exec= value from argv. argv[0] is always double-quoted; any other argument is quoted when it is
// empty (written `""`) or holds a reserved character: space, tab, CR, LF, `"`, `'`, `\`, `>`, `<`, `~`, `|`, `&`,
// `;`, `$`, `*`, `?`, `#`, `(`, `)` or "`". Inside quotes `"`, "`", `$` and `\` are backslash-escaped. Every `%`
// becomes `%%`. Then the whole line gets the string escapes, because the spec undoes those before the Exec quoting:
// `\` doubled, a tab as `\t`, CR as `\r` and LF as `\n`. Field codes (%u, %f) are appended by the caller after the
// call.
func Exec(argv ...string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		if i == 0 || a == "" || strings.ContainsAny(a, execReserved) {
			parts[i] = `"` + quotedEscaper.Replace(a) + `"`
		} else {
			parts[i] = strings.ReplaceAll(a, "%", "%%")
		}
	}
	return execLineEscaper.Replace(strings.Join(parts, " "))
}

// SplitExec is Exec's reader, for any program's entry: it undoes the string escapes, then the quoting, then `%%`,
// and returns the arguments. Every other field code (%u, %f, ...) is left in place for the caller to expand. Because
// `%%` is undone, an argument written `%%u` and a field code `%u` both read back as `%u`.
func SplitExec(value string) []string {
	s := stringUnescaper.Replace(value)
	var (
		args    []string
		arg     strings.Builder
		started bool
		quoted  bool
	)
	flush := func() {
		if started {
			args = append(args, strings.ReplaceAll(arg.String(), "%%", "%"))
		}
		arg.Reset()
		started = false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			quoted = !quoted
			started = true
		case quoted && c == '\\' && i+1 < len(s) && strings.IndexByte("\"`$\\", s[i+1]) >= 0:
			i++
			arg.WriteByte(s[i])
		case !quoted && (c == ' ' || c == '\t' || c == '\n' || c == '\r'):
			flush()
		default:
			arg.WriteByte(c)
			started = true
		}
	}
	flush()
	return args
}

// Entry is a Desktop Entry's keys in the order Marshal writes them. Each pair is a key name and its already-escaped
// value; use Value for a string key and Exec for Exec.
type Entry [][2]string

// Marshal writes a .desktop file: a [Desktop Entry] group header and one key=value line per pair, each ending in a
// newline.
func (e Entry) Marshal() string {
	var b strings.Builder
	b.WriteString("[Desktop Entry]\n")
	for _, kv := range e {
		b.WriteString(kv[0])
		b.WriteByte('=')
		b.WriteString(kv[1])
		b.WriteByte('\n')
	}
	return b.String()
}

// SetKey rewrites the first key= line in a Desktop Entry file. Every other line stays, in order, including keys
// Marshal would not write. value is the already-escaped right-hand side (see Value and Exec). A trailing newline on
// that line is kept if it was there. If the file has no such key, SetKey returns an error.
func SetKey(entry, key, value string) (string, error) {
	prefix := key + "="
	lines := strings.SplitAfter(entry, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		suffix := ""
		if strings.HasSuffix(line, "\n") {
			suffix = "\n"
		}
		lines[i] = prefix + value + suffix
		return strings.Join(lines, ""), nil
	}
	return "", errors.New("desktop entry has no " + key)
}

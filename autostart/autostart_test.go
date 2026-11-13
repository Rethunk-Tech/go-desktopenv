package autostart

import (
	"strings"
	"testing"
)

func TestQuoteProgramKeepsAQuoteInsideThePath(t *testing.T) {
	t.Parallel()
	got := quoteProgram(`C:\foo"bar\app.exe`)
	if strings.HasPrefix(got, `"C:\foo"`) {
		t.Fatalf("quote in the path closed the program: %q", got)
	}
	if !strings.HasPrefix(got, `"`) || !strings.HasSuffix(got, `"`) {
		t.Fatalf("not quoted: %q", got)
	}
	got = quoteProgram(`C:\dir\`)
	if !strings.HasSuffix(got, `\\"`) {
		t.Fatalf("trailing backslash ate the close quote: %q", got)
	}
}

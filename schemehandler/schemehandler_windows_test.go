package schemehandler

import (
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/windows/registry"
)

func deleteTree(t *testing.T, path string) {
	t.Helper()
	k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return
	}
	subs, _ := k.ReadSubKeyNames(-1)
	_ = k.Close()
	for _, s := range subs {
		deleteTree(t, path+`\`+s)
	}
	if err := registry.DeleteKey(registry.CURRENT_USER, path); err != nil {
		t.Errorf("delete %s: %v", path, err)
	}
}

func TestRegisterRestoreAndHandBack(t *testing.T) {
	root := `Software\DesktopEnvTest` + strconv.FormatInt(time.Now().UnixNano(), 36)
	t.Cleanup(func() { deleteTree(t, root) })
	other := &Handler{cfg: Config{Exe: `C:\Other\other.exe`}, classes: root}
	h := &Handler{cfg: Config{Exe: `C:\App\app.exe`, Schemes: []string{"nxm", "ror2mm"}}, classes: root}

	if err := other.Register("nxm"); err != nil {
		t.Fatal(err)
	}
	before, err := h.Owner("nxm")
	if err != nil || before.Mine || before.Name != `C:\Other\other.exe` || before.ID == "" {
		t.Fatalf("owner before: %+v, %v", before, err)
	}
	if err := h.Register("nxm", "ror2mm"); err != nil {
		t.Fatal(err)
	}
	if o, err := h.Owner("nxm"); err != nil || !o.Mine {
		t.Fatalf("owner after Register: %+v, %v", o, err)
	}
	if got := readDefault(h.commandKey("nxm")); got != `"C:\App\app.exe" "%1"` {
		t.Fatalf("command %q", got)
	}
	if err := h.Restore(map[string]string{"nxm": before.ID}); err != nil {
		t.Fatal(err)
	}
	if o, _ := h.Owner("nxm"); o != before {
		t.Errorf("nxm owner after Restore: %+v", o)
	}
	if o, _ := h.Owner("ror2mm"); o != (Owner{}) {
		t.Errorf("ror2mm kept an owner: %+v", o)
	}
	// A scheme another program owns is not this one's to hand back.
	if err := h.Release([]string{"nxm"}, nil); err != nil {
		t.Fatal(err)
	}
	if o, _ := h.Owner("nxm"); o != before {
		t.Errorf("Release removed another program's scheme: %+v", o)
	}
}

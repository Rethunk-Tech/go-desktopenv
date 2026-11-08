//go:build windows

package firewall

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
)

const (
	fwDirectionIn  = 1
	fwActionBlock  = 0
	fwActionAllow  = 1
	elevatedWaitMS = 5 * 60 * 1000
)

// policy runs f against the firewall policy object on a locked OS thread, as COM requires.
func policy(f func(policy, rules *ole.IDispatch) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		var oleErr *ole.OleError
		// S_FALSE: this thread already initialised COM.
		if !errors.As(err, &oleErr) || oleErr.Code() != 1 {
			return fmt.Errorf("initialise COM: %w", err)
		}
	}
	defer ole.CoUninitialize()
	unknown, err := oleutil.CreateObject("HNetCfg.FwPolicy2")
	if err != nil {
		return fmt.Errorf("open Windows Firewall policy: %w", err)
	}
	defer unknown.Release()
	pol, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer pol.Release()
	rulesVar, err := oleutil.GetProperty(pol, "Rules")
	if err != nil {
		return fmt.Errorf("read firewall rules: %w", err)
	}
	defer func() { _ = rulesVar.Clear() }()
	return f(pol, rulesVar.ToIDispatch())
}

func intProp(d *ole.IDispatch, name string) (int32, error) {
	v, err := oleutil.GetProperty(d, name)
	if err != nil {
		return 0, err
	}
	defer func() { _ = v.Clear() }()
	n, ok := v.Value().(int32)
	if !ok {
		return 0, fmt.Errorf("firewall property %s is not a number", name)
	}
	return n, nil
}

func strProp(d *ole.IDispatch, name string) (string, error) {
	v, err := oleutil.GetProperty(d, name)
	if err != nil {
		return "", err
	}
	defer func() { _ = v.Clear() }()
	return v.ToString(), nil
}

func readRule(d *ole.IDispatch) (Rule, error) {
	app, err := strProp(d, "ApplicationName")
	if err != nil {
		return Rule{}, err
	}
	dir, err := intProp(d, "Direction")
	if err != nil {
		return Rule{}, err
	}
	action, err := intProp(d, "Action")
	if err != nil {
		return Rule{}, err
	}
	enabled, err := oleutil.GetProperty(d, "Enabled")
	if err != nil {
		return Rule{}, err
	}
	on := enabled.Value() == true
	_ = enabled.Clear()
	profiles, err := intProp(d, "Profiles")
	if err != nil {
		return Rule{}, err
	}
	return Rule{App: app, Inbound: dir == fwDirectionIn, Allow: action == fwActionAllow, Enabled: on, Profiles: profiles}, nil
}

// CurrentProfileTypes returns the connected networks' Windows Firewall profile
// bits (NET_FW_PROFILE2_*).
func CurrentProfileTypes() (int32, error) {
	var current int32
	err := policy(func(pol, _ *ole.IDispatch) error {
		var err error
		current, err = intProp(pol, "CurrentProfileTypes")
		return err
	})
	if err != nil {
		return 0, err
	}
	return current, nil
}

// Rules returns the inbound-relevant fields of every Windows Firewall rule.
func Rules() ([]Rule, error) {
	var rules []Rule
	err := policy(func(_, set *ole.IDispatch) error {
		return oleutil.ForEach(set, func(v *ole.VARIANT) error {
			r, err := readRule(v.ToIDispatch())
			if err != nil {
				return err
			}
			rules = append(rules, r)
			return nil
		})
	})
	if err != nil {
		return nil, fmt.Errorf("read Windows Firewall rules: %w", err)
	}
	return rules, nil
}

// Add removes every inbound block rule for exe (Windows names the ones it
// creates itself after the file description) and every rule for exe named name
// or publicName, then writes an inbound allow rule on every network profile
// when anyAddr, or an allow on Domain and Private with an explicit Public
// block so Windows never prompts.
func Add(exe, name, publicName string, anyAddr bool) error {
	return policy(func(_, set *ole.IDispatch) error {
		// Rules are removed by name, and a name can be shared with another program's rule, so each match is first
		// renamed to a token of its own.
		var tokens []string
		err := oleutil.ForEach(set, func(v *ole.VARIANT) error {
			d := v.ToIDispatch()
			r, err := readRule(d)
			if err != nil {
				return err
			}
			ruleName, err := strProp(d, "Name")
			if err != nil {
				return err
			}
			inboundBlock := r.Inbound && !r.Allow
			if !strings.EqualFold(r.App, exe) || (!inboundBlock && ruleName != name && ruleName != publicName) {
				return nil
			}
			token := "remove-" + strconv.Itoa(len(tokens))
			if _, err := oleutil.PutProperty(d, "Name", token); err != nil {
				return err
			}
			tokens = append(tokens, token)
			return nil
		})
		if err != nil {
			return err
		}
		for _, token := range tokens {
			if _, err := oleutil.CallMethod(set, "Remove", token); err != nil {
				return fmt.Errorf("remove firewall rule: %w", err)
			}
		}
		if anyAddr {
			return addRule(set, name, exe, fwActionAllow, ProfilesAll)
		}
		if err := addRule(set, name, exe, fwActionAllow, ProfilesLocal); err != nil {
			return err
		}
		return addRule(set, publicName, exe, fwActionBlock, ProfilePublic)
	})
}

func addRule(set *ole.IDispatch, name, exe string, action, profiles int32) error {
	unknown, err := oleutil.CreateObject("HNetCfg.FWRule")
	if err != nil {
		return err
	}
	defer unknown.Release()
	rule, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer rule.Release()
	if _, err := oleutil.PutProperty(rule, "Name", name); err != nil {
		return fmt.Errorf("set firewall rule Name: %w", err)
	}
	if _, err := oleutil.PutProperty(rule, "ApplicationName", exe); err != nil {
		return fmt.Errorf("set firewall rule ApplicationName: %w", err)
	}
	if _, err := oleutil.PutProperty(rule, "Direction", int32(fwDirectionIn)); err != nil {
		return fmt.Errorf("set firewall rule Direction: %w", err)
	}
	if _, err := oleutil.PutProperty(rule, "Action", action); err != nil {
		return fmt.Errorf("set firewall rule Action: %w", err)
	}
	if _, err := oleutil.PutProperty(rule, "Profiles", profiles); err != nil {
		return fmt.Errorf("set firewall rule Profiles: %w", err)
	}
	if _, err := oleutil.PutProperty(rule, "Enabled", true); err != nil {
		return fmt.Errorf("set firewall rule Enabled: %w", err)
	}
	_, err = oleutil.CallMethod(set, "Add", rule)
	return err
}

// shellExecuteInfo is SHELLEXECUTEINFOW.
type shellExecuteInfo struct {
	Size       uint32
	Mask       uint32
	Hwnd       windows.Handle
	Verb       *uint16
	File       *uint16
	Parameters *uint16
	Directory  *uint16
	Show       int32
	Instance   windows.Handle
	IDList     uintptr
	Class      *uint16
	KeyClass   windows.Handle
	HotKey     uint32
	Icon       windows.Handle
	Process    windows.Handle
}

const seeMaskNoCloseProcess = 0x40

var shellExecuteEx = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW")

// Elevate starts exe with args via ShellExecuteExW runas, so the UAC prompt
// names that program and its publisher, and waits for its exit code. Declining
// the prompt is an error.
func Elevate(exe, args string) error {
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	params, err := windows.UTF16PtrFromString(args)
	if err != nil {
		return err
	}
	info := shellExecuteInfo{Mask: seeMaskNoCloseProcess, Verb: verb, File: file, Parameters: params}
	info.Size = uint32(unsafe.Sizeof(info))
	if r, _, callErr := shellExecuteEx.Call(uintptr(unsafe.Pointer(&info))); r == 0 { //nolint:gosec // G103: ShellExecuteEx takes a pointer to the info struct
		return fmt.Errorf("start the elevated firewall step: %w", callErr)
	}
	defer func() { _ = windows.CloseHandle(info.Process) }()
	if ev, err := windows.WaitForSingleObject(info.Process, elevatedWaitMS); err != nil || ev != windows.WAIT_OBJECT_0 {
		return errors.New("the elevated firewall step did not finish")
	}
	var code uint32
	if err := windows.GetExitCodeProcess(info.Process, &code); err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("update Windows Firewall rule: exit status %d", code)
	}
	return nil
}

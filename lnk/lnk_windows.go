//go:build windows

package lnk

import (
	"errors"
	"runtime"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// withLink runs fn on the shortcut object for path. COM needs one OS thread for the whole call.
func withLink(path string, fn func(*ole.IDispatch) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		var oleErr *ole.OleError
		// S_FALSE means COM was already initialised on this thread, which is fine.
		if !errors.As(err, &oleErr) || oleErr.Code() != 1 {
			return err
		}
	}
	defer ole.CoUninitialize()
	unknown, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return err
	}
	defer unknown.Release()
	shell, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return err
	}
	defer shell.Release()
	linkVar, err := oleutil.CallMethod(shell, "CreateShortcut", path)
	if err != nil {
		return err
	}
	link := linkVar.ToIDispatch()
	defer link.Release()
	return fn(link)
}

// Write saves s as the .lnk at path, replacing any file there. The target and arguments are always set, so an
// empty Arguments clears the old ones.
func Write(path string, s Shortcut) error {
	props := map[string]string{"TargetPath": s.Target, "Arguments": s.Arguments}
	for prop, value := range map[string]string{
		"WorkingDirectory": s.WorkingDir, "IconLocation": s.Icon, "Description": s.Description,
	} {
		if value != "" {
			props[prop] = value
		}
	}
	return withLink(path, func(link *ole.IDispatch) error {
		for prop, value := range props {
			if _, err := oleutil.PutProperty(link, prop, value); err != nil {
				return err
			}
		}
		_, err := oleutil.CallMethod(link, "Save")
		return err
	})
}

// Arguments is the command-line arguments stored in the .lnk at path.
func Arguments(path string) (string, error) {
	var arguments string
	err := withLink(path, func(link *ole.IDispatch) error {
		value, err := oleutil.GetProperty(link, "Arguments")
		if err != nil {
			return err
		}
		defer func() { _ = value.Clear() }()
		arguments = value.ToString()
		return nil
	})
	return arguments, err
}

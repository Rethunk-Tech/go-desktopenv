// Package firewall reads Windows Firewall profile types and inbound rules for a
// program, adds an inbound allow rule, and self-elevates that add through
// ShellExecuteExW runas. CurrentProfileTypes, Rules, Add and Elevate return
// ErrUnsupported on operating systems other than Windows.
package firewall

import (
	"errors"
	"strings"
)

// ErrUnsupported is returned by CurrentProfileTypes, Rules, Add and Elevate
// when the operating system has no Windows Firewall COM API.
var ErrUnsupported = errors.New("firewall: unsupported")

// Windows Firewall profile bits (NET_FW_PROFILE2_*).
const (
	// ProfileDomain is NET_FW_PROFILE2_DOMAIN.
	ProfileDomain int32 = 1
	// ProfilePrivate is NET_FW_PROFILE2_PRIVATE.
	ProfilePrivate int32 = 2
	// ProfilePublic is NET_FW_PROFILE2_PUBLIC.
	ProfilePublic int32 = 4
	// ProfilesAll is the rule value that matches every profile (all bits set).
	ProfilesAll int32 = 0x7FFFFFFF
	// ProfilesLocal is Domain and Private, the profiles an allow rule covers
	// when Public is blocked explicitly so Windows never prompts.
	ProfilesLocal = ProfileDomain | ProfilePrivate
	// ProfilesEvery is Domain, Private and Public.
	ProfilesEvery = ProfilesLocal | ProfilePublic
)

// Rule is the part of a Windows Firewall rule that decides whether a program
// is reachable inbound.
type Rule struct {
	// App is the rule's ApplicationName.
	App string
	// Inbound reports whether Direction is inbound.
	Inbound bool
	// Allow reports whether Action is allow.
	Allow bool
	// Enabled reports whether the rule is on.
	Enabled bool
	// Profiles is the rule's NET_FW_PROFILE2_* bit mask.
	Profiles int32
}

func (r Rule) forExe(exe string) bool {
	return r.Enabled && r.Inbound && strings.EqualFold(r.App, exe)
}

func (r Rule) overlaps(profiles int32) bool {
	return r.Profiles == ProfilesAll || r.Profiles&profiles != 0
}

func (r Rule) includes(profiles int32) bool {
	return r.Profiles == ProfilesAll || r.Profiles&profiles == profiles
}

// OnlyPublic reports that the connected networks' profile bits
// (CurrentProfileTypes) name Public and neither Domain nor Private.
func OnlyPublic(current int32) bool {
	return current&ProfilePublic != 0 && current&ProfilesLocal == 0
}

// Blocked reports whether rules are not yet the set that lets exe accept
// inbound connections without Windows asking: an allow rule on every profile
// it should be reachable on, no block rule on those, and, when anyAddr is
// false, a block rule on Public so every profile has a matching rule.
func Blocked(rules []Rule, exe string, anyAddr bool) bool {
	reach := ProfilesLocal
	if anyAddr {
		reach = ProfilesEvery
	}
	allowed, publicBlocked := false, false
	for _, r := range rules {
		if !r.forExe(exe) {
			continue
		}
		switch {
		case r.Allow && r.includes(reach):
			allowed = true
		case !r.Allow && r.overlaps(reach):
			return true
		case !r.Allow && r.includes(ProfilePublic):
			publicBlocked = true
		}
	}
	return !allowed || (!anyAddr && !publicBlocked)
}

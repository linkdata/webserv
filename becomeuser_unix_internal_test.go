//go:build unix || linux

package webserv

import (
	"errors"
	"os/user"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

type becomeUserFns struct {
	lookupUserFn func(username string) (*user.User, error)
	groupIDsFn   func(u *user.User) ([]string, error)
	geteuidFn    func() int
	getegidFn    func() int
	setgroupsFn  func(gids []int) error
	setgidFn     func(gid int) error
	setuidFn     func(uid int) error
	unsetenvFn   func(key string) error
	setenvFn     func(key, value string) error
}

func captureBecomeUserFns() becomeUserFns {
	return becomeUserFns{
		lookupUserFn: lookupUserFn,
		groupIDsFn:   groupIDsFn,
		geteuidFn:    geteuidFn,
		getegidFn:    getegidFn,
		setgroupsFn:  setgroupsFn,
		setgidFn:     setgidFn,
		setuidFn:     setuidFn,
		unsetenvFn:   unsetenvFn,
		setenvFn:     setenvFn,
	}
}

func restoreBecomeUserFns(fns becomeUserFns) {
	lookupUserFn = fns.lookupUserFn
	groupIDsFn = fns.groupIDsFn
	geteuidFn = fns.geteuidFn
	getegidFn = fns.getegidFn
	setgroupsFn = fns.setgroupsFn
	setgidFn = fns.setgidFn
	setuidFn = fns.setuidFn
	unsetenvFn = fns.unsetenvFn
	setenvFn = fns.setenvFn
}

func TestBecomeUser_RootSetsSupplementaryGroupsBeforeDroppingPrivileges(t *testing.T) {
	saved := captureBecomeUserFns()
	defer restoreBecomeUserFns(saved)

	u := &user.User{Username: "svc", Uid: "101", Gid: "0", HomeDir: "/tmp/svc"}
	sequence := make([]string, 0, 8)
	gotGroups := []int(nil)

	lookupUserFn = func(username string) (*user.User, error) {
		sequence = append(sequence, "lookup")
		if username != "svc" {
			t.Fatalf("lookup user %q", username)
		}
		return u, nil
	}
	groupIDsFn = func(_ *user.User) ([]string, error) {
		sequence = append(sequence, "groupids")
		return []string{"0", "202"}, nil
	}
	geteuidFn = func() int { return 0 }
	setgroupsFn = func(gids []int) error {
		sequence = append(sequence, "setgroups")
		gotGroups = append([]int{}, gids...)
		return nil
	}
	setgidFn = func(gid int) error {
		sequence = append(sequence, "setgid")
		if gid != 0 {
			t.Fatalf("gid=%d", gid)
		}
		return nil
	}
	setuidFn = func(uid int) error {
		sequence = append(sequence, "setuid")
		if uid != 101 {
			t.Fatalf("uid=%d", uid)
		}
		return nil
	}
	unsetenvFn = func(key string) error {
		sequence = append(sequence, "unsetenv:"+key)
		return nil
	}
	setenvFn = func(key, value string) error {
		sequence = append(sequence, "setenv:"+key+"="+value)
		return nil
	}

	if err := BecomeUser("svc"); err != nil {
		t.Fatal(err)
	}

	wantGroups := []int{0, 202}
	if !reflect.DeepEqual(gotGroups, wantGroups) {
		t.Fatalf("groups=%v want %v", gotGroups, wantGroups)
	}

	want := []string{
		"lookup",
		"groupids",
		"setgroups",
		"setgid",
		"setuid",
		"unsetenv:XDG_CONFIG_HOME",
		"setenv:HOME=/tmp/svc",
		"setenv:USER=svc",
	}
	if !reflect.DeepEqual(sequence, want) {
		t.Fatalf("sequence=%v want %v", sequence, want)
	}
}

func TestBecomeUser_NonRootSkipsSupplementaryGroups(t *testing.T) {
	saved := captureBecomeUserFns()
	defer restoreBecomeUserFns(saved)

	u := &user.User{Username: "svc", Uid: "1000", Gid: "1000", HomeDir: "/tmp/svc"}
	groupIDsCalled := false
	setgroupsCalled := false

	lookupUserFn = func(_ string) (*user.User, error) { return u, nil }
	groupIDsFn = func(_ *user.User) ([]string, error) {
		groupIDsCalled = true
		return []string{"1000"}, nil
	}
	geteuidFn = func() int { return 1000 }
	getegidFn = func() int { return 1000 }
	setgroupsFn = func(_ []int) error {
		setgroupsCalled = true
		return nil
	}
	gotGid, gotUid := -1, -1
	setgidFn = func(gid int) error {
		gotGid = gid
		return nil
	}
	setuidFn = func(uid int) error {
		gotUid = uid
		return nil
	}
	unsetenvFn = func(_ string) error { return nil }
	setenvFn = func(_, _ string) error { return nil }

	if err := BecomeUser("svc"); err != nil {
		t.Fatal(err)
	}
	if gotGid != 1000 || gotUid != 1000 {
		t.Fatalf("setgid(%d) setuid(%d), want 1000 for both", gotGid, gotUid)
	}
	if groupIDsCalled {
		t.Fatal("groupIDsFn called for non-root user")
	}
	if setgroupsCalled {
		t.Fatal("setgroupsFn called for non-root user")
	}
}

func TestBecomeUser_NonRootRefusesIdentityChange(t *testing.T) {
	tests := []struct {
		name       string
		euid, egid int
		uid, gid   string
	}{
		{"uid differs", 1001, 1000, "1000", "1000"},
		{"gid differs", 1000, 1001, "1000", "1000"},
		{"target is root", 1000, 1000, "0", "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved := captureBecomeUserFns()
			defer restoreBecomeUserFns(saved)

			u := &user.User{Username: "svc", Uid: tt.uid, Gid: tt.gid, HomeDir: "/tmp/svc"}
			called := false

			lookupUserFn = func(_ string) (*user.User, error) { return u, nil }
			groupIDsFn = func(_ *user.User) ([]string, error) { called = true; return nil, nil }
			geteuidFn = func() int { return tt.euid }
			getegidFn = func() int { return tt.egid }
			setgroupsFn = func(_ []int) error { called = true; return nil }
			setgidFn = func(_ int) error { called = true; return nil }
			setuidFn = func(_ int) error { called = true; return nil }
			unsetenvFn = func(_ string) error { called = true; return nil }
			setenvFn = func(_, _ string) error { called = true; return nil }

			err := BecomeUser("svc")
			if !errors.Is(err, ErrBecomeUser) || !errors.Is(err, syscall.EPERM) {
				t.Fatalf("err=%v, want ErrBecomeUser wrapping EPERM", err)
			}
			if called {
				t.Fatal("identity or environment changed after refusal")
			}
		})
	}
}

func TestBecomeUser_RootRefusesUnsafeIDs(t *testing.T) {
	tests := []struct {
		name, uid, gid string
		groups         []string
		want           string
	}{
		{"root", "0", "0", nil, "UID 0"},
		{"root alias", "00", "33", nil, "UID 0"},
		{"UID sentinel", "4294967295", "33", nil, "4294967295"},
		{"UID overflow", "4294967296", "33", nil, "4294967296"},
		{"negative UID", "-1", "33", nil, "UID -1"},
		{"GID sentinel", "33", "4294967295", nil, "4294967295"},
		{"GID overflow", "33", "4294967296", nil, "4294967296"},
		{"negative GID", "33", "-1", nil, "GID -1"},
		{"supplementary GID sentinel", "33", "33", []string{"4294967295"}, "4294967295"},
		{"supplementary GID overflow", "33", "33", []string{"50", "4294967296"}, "4294967296"},
		{"negative supplementary GID", "33", "33", []string{"-1"}, "GID -1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saved := captureBecomeUserFns()
			defer restoreBecomeUserFns(saved)

			lookupUserFn = func(_ string) (*user.User, error) {
				return &user.User{Username: "www-data", Uid: tt.uid, Gid: tt.gid}, nil
			}
			geteuidFn = func() int { return 0 }
			groupIDsCalled, changed := false, false
			groupIDsFn = func(_ *user.User) ([]string, error) {
				groupIDsCalled = true
				return tt.groups, nil
			}
			setgroupsFn = func(_ []int) error { changed = true; return nil }
			setgidFn = func(_ int) error { changed = true; return nil }
			setuidFn = func(_ int) error { changed = true; return nil }
			unsetenvFn = func(_ string) error { changed = true; return nil }
			setenvFn = func(_, _ string) error { changed = true; return nil }

			err := BecomeUser("www-data")
			if !errors.Is(err, ErrBecomeUser) || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want ErrBecomeUser containing %q", err, tt.want)
			}
			if tt.uid == "0" || tt.uid == "00" {
				if !strings.Contains(err.Error(), "root") {
					t.Fatalf("error = %v, want reason for refusing root", err)
				}
			}
			if changed || groupIDsCalled != (tt.groups != nil) {
				t.Fatal("unexpected group lookup or credential/environment change after refusal")
			}
		})
	}
}

func TestBecomeUser_GroupIDParseErrorStopsBeforeSetgid(t *testing.T) {
	saved := captureBecomeUserFns()
	defer restoreBecomeUserFns(saved)

	u := &user.User{Username: "svc", Uid: "101", Gid: "201", HomeDir: "/tmp/svc"}
	setgidCalled := false
	setuidCalled := false

	lookupUserFn = func(_ string) (*user.User, error) { return u, nil }
	groupIDsFn = func(_ *user.User) ([]string, error) { return []string{"bad"}, nil }
	geteuidFn = func() int { return 0 }
	setgroupsFn = func(_ []int) error { return nil }
	setgidFn = func(_ int) error {
		setgidCalled = true
		return nil
	}
	setuidFn = func(_ int) error {
		setuidCalled = true
		return nil
	}
	unsetenvFn = func(_ string) error { return nil }
	setenvFn = func(_, _ string) error { return nil }

	err := BecomeUser("svc")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrBecomeUser) {
		t.Fatalf("unexpected error type: %v", err)
	}
	if setgidCalled || setuidCalled {
		t.Fatalf("setgid/setuid should not have been called, got setgid=%v setuid=%v", setgidCalled, setuidCalled)
	}
}

func TestGroupIDsFn_DefaultImplementation(t *testing.T) {
	saved := captureBecomeUserFns()
	defer restoreBecomeUserFns(saved)

	var u *user.User
	var err error
	if u, err = user.Current(); err == nil {
		var want []string
		if want, err = u.GroupIds(); err == nil {
			var got []string
			if got, err = groupIDsFn(u); err == nil {
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("groupIDsFn() = %v, want %v", got, want)
				}
			}
		}
	}
	if err != nil {
		t.Skipf("unable to resolve current user groups for default groupIDsFn test: %v", err)
	}
}

func TestParseGroupIDs_ErrorBeforeValidEntryIsNotSilentlyDropped(t *testing.T) {
	// A parse error must be reported even when a later entry parses cleanly:
	// ["bad", "201"] returns an error, not ([201], nil).
	gids, err := parseGroupIDs([]string{"bad", "201"})
	if err == nil {
		t.Fatalf("expected error for invalid group ID, got gids=%v", gids)
	}
}

func TestParseGroupIDs_Signed32BitHighGID(t *testing.T) {
	if strconv.IntSize != 32 {
		t.Skip("32-bit cgo group ID representation")
	}
	gids, err := parseGroupIDs([]string{"-2147483648"})
	if err != nil || !reflect.DeepEqual(gids, []int{-2147483648}) {
		t.Fatalf("gids=%v, err=%v", gids, err)
	}
}

func TestBecomeUser_RootWithoutSupplementaryGroupsFallsBackToPrimaryGroup(t *testing.T) {
	saved := captureBecomeUserFns()
	defer restoreBecomeUserFns(saved)

	u := &user.User{Username: "svc", Uid: "101", Gid: "201", HomeDir: "/tmp/svc"}
	gotGroups := []int(nil)

	lookupUserFn = func(_ string) (*user.User, error) { return u, nil }
	groupIDsFn = func(_ *user.User) ([]string, error) { return []string{}, nil }
	geteuidFn = func() int { return 0 }
	setgroupsFn = func(gids []int) error {
		gotGroups = append([]int{}, gids...)
		return nil
	}
	setgidFn = func(gid int) error {
		if gid != 201 {
			t.Fatalf("gid=%d", gid)
		}
		return nil
	}
	setuidFn = func(_ int) error { return nil }
	unsetenvFn = func(_ string) error { return nil }
	setenvFn = func(_, _ string) error { return nil }

	if err := BecomeUser("svc"); err != nil {
		t.Fatal(err)
	}
	wantGroups := []int{201}
	if !reflect.DeepEqual(gotGroups, wantGroups) {
		t.Fatalf("groups=%v want %v", gotGroups, wantGroups)
	}
}

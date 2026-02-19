//go:build unix || linux

package webserv

import (
	"errors"
	"os/user"
	"reflect"
	"testing"
)

type becomeUserFns struct {
	lookupUserFn func(username string) (*user.User, error)
	groupIDsFn   func(u *user.User) ([]string, error)
	geteuidFn    func() int
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
	setgroupsFn = fns.setgroupsFn
	setgidFn = fns.setgidFn
	setuidFn = fns.setuidFn
	unsetenvFn = fns.unsetenvFn
	setenvFn = fns.setenvFn
}

func TestBecomeUser_RootSetsSupplementaryGroupsBeforeDroppingPrivileges(t *testing.T) {
	saved := captureBecomeUserFns()
	defer restoreBecomeUserFns(saved)

	u := &user.User{Username: "svc", Uid: "101", Gid: "201", HomeDir: "/tmp/svc"}
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
		return []string{"201", "202"}, nil
	}
	geteuidFn = func() int { return 0 }
	setgroupsFn = func(gids []int) error {
		sequence = append(sequence, "setgroups")
		gotGroups = append([]int{}, gids...)
		return nil
	}
	setgidFn = func(gid int) error {
		sequence = append(sequence, "setgid")
		if gid != 201 {
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

	wantGroups := []int{201, 202}
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
	setgroupsFn = func(_ []int) error {
		setgroupsCalled = true
		return nil
	}
	setgidFn = func(_ int) error { return nil }
	setuidFn = func(_ int) error { return nil }
	unsetenvFn = func(_ string) error { return nil }
	setenvFn = func(_, _ string) error { return nil }

	if err := BecomeUser("svc"); err != nil {
		t.Fatal(err)
	}
	if groupIDsCalled {
		t.Fatal("groupIDsFn called for non-root user")
	}
	if setgroupsCalled {
		t.Fatal("setgroupsFn called for non-root user")
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


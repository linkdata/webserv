package webserv_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/linkdata/webserv"
)

func TestBecomeUser(t *testing.T) {
	if err := webserv.BecomeUser(""); err != nil {
		t.Error(err)
	}
	const noSuchUser = "no-such-user"
	if err := webserv.BecomeUser(noSuchUser); err == nil {
		t.Error(noSuchUser)
	} else if !errors.Is(err, webserv.ErrBecomeUser) {
		t.Errorf("expected errors.Is(err, ErrBecomeUser), got: %v", err)
	}
}

func Test_ErrBecomeUser(t *testing.T) {
	err := webserv.BecomeUser("!no!")
	t.Logf("%#v: %q", err, err.Error())
	if err == nil {
		t.Error("expected error")
	}
	if !errors.Is(err, webserv.ErrBecomeUser) {
		t.Errorf("%#v", err)
	}
	if errors.Unwrap(err) == nil {
		t.Error("Unwrap was nil")
	}
	if !strings.HasPrefix(err.Error(), "BecomeUser") {
		t.Error("missing prefix")
	}
}

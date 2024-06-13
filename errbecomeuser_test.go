package webserv_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/linkdata/webserv"
)

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

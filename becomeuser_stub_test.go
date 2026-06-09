//go:build !(unix || linux)

package webserv_test

import (
	"errors"
	"testing"

	"github.com/linkdata/webserv"
)

// TestBecomeUser_UnsupportedOS verifies the documented contract of the
// non-unix stub: a non-empty user name yields an error matching both
// [webserv.ErrBecomeUser] and [errors.ErrUnsupported], while an empty user
// name is a no-op returning nil.
func TestBecomeUser_UnsupportedOS(t *testing.T) {
	err := webserv.BecomeUser("someuser")
	if err == nil {
		t.Fatal("expected error on unsupported OS")
	}
	if !errors.Is(err, webserv.ErrBecomeUser) {
		t.Errorf("error %v does not match webserv.ErrBecomeUser", err)
	}
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("error %v does not match errors.ErrUnsupported", err)
	}

	if err := webserv.BecomeUser(""); err != nil {
		t.Errorf("BecomeUser(%q) = %v, want nil", "", err)
	}
}

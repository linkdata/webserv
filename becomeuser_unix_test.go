//go:build unix || linux

package webserv_test

import (
	"os"
	"os/user"
	"strconv"
	"testing"

	"github.com/linkdata/webserv"
)

// TestBecomeUser_CurrentUserSucceeds verifies that switching to the user the
// process already runs as is a no-op that succeeds. Privilege switching is only
// supported on unix; see becomeuser_test.go for the cross-platform cases.
func TestBecomeUser_CurrentUserSucceeds(t *testing.T) {
	if userName := os.Getenv("USER"); userName != "" {
		u, err := user.Lookup(userName)
		if err != nil {
			t.Fatal(err)
		}
		if u.Uid != strconv.Itoa(os.Geteuid()) || u.Gid != strconv.Itoa(os.Getegid()) {
			t.Skip("target user does not match effective UID and GID")
		}
		if err := webserv.BecomeUser(userName); err != nil {
			t.Error(err)
		}
	}
}

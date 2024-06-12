//go:build !(unix || linux)

package webserv

import (
	"errors"
	"runtime"
)

var ErrBecomeUserNotImplemented = errors.New("user switching not implemented for " + runtime.GOOS)

// BecomeUser switches to the given userName if not empty.
//
// It sets the GID, UID and changes the USER and HOME
// environment variables accordingly. It unsets XDG_CONFIG_HOME.
//
// Returns ErrBecomeUserNotImplemented if the current OS is not supported.
func BecomeUser(userName string) (err error) {
	if userName != "" {
		err = ErrBecomeUserNotImplemented
	}
	return
}

//go:build !(unix || linux)

package webserv

import (
	"errors"
)

// BecomeUser switches to the given userName if not empty.
//
// It sets the GID, UID and changes the USER and HOME
// environment variables accordingly. It unsets XDG_CONFIG_HOME.
//
// Returns an error matching both ErrBecomeUser and errors.ErrUnsupported
// if the current OS is not supported.
func BecomeUser(userName string) (err error) {
	if userName != "" {
		err = newErrBecomeUser(userName, errors.ErrUnsupported)
	}
	return
}

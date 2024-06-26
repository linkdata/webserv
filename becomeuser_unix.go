//go:build unix || linux

package webserv

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// BecomeUser switches to the given userName if not empty.
//
// It sets the GID, UID and changes the USER and HOME
// environment variables accordingly. It unsets XDG_CONFIG_HOME.
//
// Returns ErrBecomeUserNotImplemented if the current OS is not supported.
func BecomeUser(userName string) error {
	var err error
	if userName != "" {
		var u *user.User
		if u, err = user.Lookup(userName); err == nil {
			var uid, gid int
			if uid, err = strconv.Atoi(u.Uid); err == nil {
				if gid, err = strconv.Atoi(u.Gid); err == nil {
					if err = syscall.Setgid(gid); err == nil {
						if err = syscall.Setuid(uid); err == nil {
							_ = os.Unsetenv("XDG_CONFIG_HOME")
							if err = os.Setenv("HOME", u.HomeDir); err == nil {
								err = os.Setenv("USER", u.Username)
							}
						}
					}
				}
			}
		}
	}
	return newErrBecomeUser(userName, err)
}

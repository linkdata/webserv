//go:build unix || linux

package webserv

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

func BecomeUser(userName string) (err error) {
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
	return
}

//go:build unix || linux

package webserv

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

var (
	lookupUserFn = user.Lookup
	groupIDsFn   = func(u *user.User) ([]string, error) { return u.GroupIds() }
	geteuidFn    = os.Geteuid
	setgroupsFn  = syscall.Setgroups
	setgidFn     = syscall.Setgid
	setuidFn     = syscall.Setuid
	unsetenvFn   = os.Unsetenv
	setenvFn     = os.Setenv
)

func parseGroupIDs(groupIDs []string) (gids []int, err error) {
	gids = make([]int, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		var gid int
		if gid, err = strconv.Atoi(groupID); err == nil {
			gids = append(gids, gid)
		}
	}
	return
}

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
		if u, err = lookupUserFn(userName); err == nil {
			var uid, gid int
			if uid, err = strconv.Atoi(u.Uid); err == nil {
				if gid, err = strconv.Atoi(u.Gid); err == nil {
					if geteuidFn() == 0 {
						var groupIDs []string
						if groupIDs, err = groupIDsFn(u); err == nil {
							var gids []int
							if gids, err = parseGroupIDs(groupIDs); err == nil {
								if len(gids) == 0 {
									gids = []int{gid}
								}
								err = setgroupsFn(gids)
							}
						}
					}
					if err == nil {
						if err = setgidFn(gid); err == nil {
							if err = setuidFn(uid); err == nil {
								_ = unsetenvFn("XDG_CONFIG_HOME")
								if err = setenvFn("HOME", u.HomeDir); err == nil {
									err = setenvFn("USER", u.Username)
								}
							}
						}
					}
				}
			}
		}
	}
	return newErrBecomeUser(userName, err)
}

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
	for i := 0; i < len(groupIDs) && err == nil; i++ {
		var gid int
		if gid, err = strconv.Atoi(groupIDs[i]); err == nil {
			gids = append(gids, gid)
		}
	}
	return
}

// BecomeUser switches to the given userName if not empty.
//
// When running as root (euid 0) it resets the supplementary groups to those of
// the target user and then sets the GID and UID, in that order (setgroups,
// setgid, setuid). When not root the supplementary-groups reset is skipped and
// setgid/setuid will fail unless the target ids already match the process.
//
// It then sets the HOME and USER environment variables to match the target user
// and unsets XDG_CONFIG_HOME so config-directory lookups follow the new HOME.
//
// Returns an error matching [ErrBecomeUser] on failure.
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

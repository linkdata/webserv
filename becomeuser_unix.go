//go:build unix || linux

package webserv

import (
	"fmt"
	"math"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

var (
	lookupUserFn = user.Lookup
	groupIDsFn   = func(u *user.User) ([]string, error) { return u.GroupIds() }
	geteuidFn    = os.Geteuid
	getegidFn    = os.Getegid
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
			// On 32-bit cgo, os/user formats high uint32 GIDs as negative ints.
			if gid == -1 || (strconv.IntSize == 64 && (gid < 0 || uint64(gid) >= math.MaxUint32)) {
				err = fmt.Errorf("supplementary GID %d is outside the valid range 0..%d", gid, uint64(math.MaxUint32-1))
			} else {
				gids = append(gids, gid)
			}
		} else {
			err = fmt.Errorf("supplementary GID %q: %w", groupIDs[i], err)
		}
	}
	return
}

// BecomeUser switches to the given userName if not empty.
//
// When running as root (euid 0), it refuses targets resolving to UID 0 or
// invalid IDs before changing credentials. It then resets supplementary groups
// to those of the target user and sets the GID and UID, in that order
// (setgroups, setgid, setuid). When not root the groups reset is skipped and
// it fails with [syscall.EPERM] unless the target UID and GID already match
// the effective ids of the process.
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
					euid := geteuidFn()
					if euid == 0 {
						switch {
						case uid == 0:
							err = fmt.Errorf("resolved UID %d would leave the process running as root", uid)
						case uid < 0 || uint64(uid) >= math.MaxUint32:
							err = fmt.Errorf("resolved UID %d is outside the valid range 1..%d", uid, uint64(math.MaxUint32-1))
						case gid < 0 || uint64(gid) >= math.MaxUint32:
							err = fmt.Errorf("resolved GID %d is outside the valid range 0..%d", gid, uint64(math.MaxUint32-1))
						default:
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
					} else if uid != euid || gid != getegidFn() {
						// Without euid 0, changing IDs can retain supplementary
						// groups or privileges, or be reversible through real/saved
						// IDs. Matching effective IDs are allowed, but any existing
						// saved IDs or privileges remain out of scope.
						err = syscall.EPERM
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
				} else {
					err = fmt.Errorf("resolved GID %q: %w", u.Gid, err)
				}
			} else {
				err = fmt.Errorf("resolved UID %q: %w", u.Uid, err)
			}
		}
	}
	return newErrBecomeUser(userName, err)
}

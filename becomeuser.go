//go:build !unix

package webserv

import (
	"errors"
	"runtime"
)

var ErrBecomeUserNotImplemented = errors.New("user switching not implemented for " + runtime.GOOS)

func BecomeUser(userName string) error {
	return ErrBecomeUserNotImplemented
}

package webserv

import "fmt"

type errBecomeUser struct {
	userName string
	err      error
}

var ErrBecomeUser = errBecomeUser{}

func (e errBecomeUser) Error() string {
	return fmt.Sprintf("BecomeUser(\"%s\"): %v", e.userName, e.err)
}

func (e errBecomeUser) Is(other error) (yes bool) {
	_, yes = other.(errBecomeUser)
	return
}

func (e errBecomeUser) Unwrap() error {
	return e.err
}

func newErrBecomeUser(userName string, err error) error {
	if err != nil {
		err = errBecomeUser{userName: userName, err: err}
	}
	return err
}

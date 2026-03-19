package webserv_test

import (
	"errors"
	"os"
	"testing"

	"github.com/linkdata/webserv"
)

func TestBecomeUser(t *testing.T) {
	if err := webserv.BecomeUser(""); err != nil {
		t.Error(err)
	}
	if userName := os.Getenv("USER"); userName != "" {
		if err := webserv.BecomeUser(userName); err != nil {
			t.Error(err)
		}
	}
	const noSuchUser = "no-such-user"
	if err := webserv.BecomeUser(noSuchUser); err == nil {
		t.Error(noSuchUser)
	} else if !errors.Is(err, webserv.ErrBecomeUser) {
		t.Errorf("expected errors.Is(err, ErrBecomeUser), got: %v", err)
	}
}

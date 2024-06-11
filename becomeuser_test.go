package webserv_test

import (
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
	}
}

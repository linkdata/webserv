package webserv_test

import (
	"strings"
	"testing"

	"github.com/linkdata/webserv"
)

func TestDefaultDataDir(t *testing.T) {
	got, err := webserv.DefaultDataDir("", "suffix")
	if err != nil {
		t.Error(err)
	}
	if !strings.HasSuffix(got, "suffix") {
		t.Error(got)
	}

	got, err = webserv.DefaultDataDir("foo", "suffix")
	if err != nil {
		t.Error(err)
	}
	if got != "foo" {
		t.Error(got)
	}
}

func TestUseDataDir(t *testing.T) {
	got, err := webserv.UseDataDir(".")
	if err != nil {
		t.Error(err)
	}
	if got == "" || got == "." {
		t.Error("failed to expand path")
	}
}

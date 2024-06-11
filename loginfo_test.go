package webserv_test

import (
	"bytes"
	"testing"

	"github.com/linkdata/webserv"
)

type printfer struct {
	called bool
}

func (x *printfer) Printf(msg string, args ...any) {
	x.called = true
}

type infoer struct {
	called bool
}

func (x *infoer) Info(msg string, args ...any) {
	x.called = true
}

func TestLogInfo(t *testing.T) {
	a := printfer{}
	webserv.LogInfo(&a, "a")
	if !a.called {
		t.Error("printfer failed")
	}

	b := infoer{}
	webserv.LogInfo(&b, "a")
	if !b.called {
		t.Error("infoer failed")
	}

	var c bytes.Buffer
	webserv.LogInfo(&c, "c")
	if c.Len() != 1 {
		t.Error("io.Writer failed")
	}
}

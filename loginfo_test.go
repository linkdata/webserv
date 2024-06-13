package webserv_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/linkdata/webserv"
)

type printlner struct {
	str string
}

func (x *printlner) Println(args ...any) {
	x.str += fmt.Sprint(args...)
}

type infoer struct {
	str string
}

func (x *infoer) Info(msg string, args ...any) {
	x.str += msg
}

func TestLogInfo(t *testing.T) {
	a := printlner{}
	webserv.LogInfo(&a, "a")
	webserv.LogInfo(&a, "\na\n")
	if a.str != "a\na" {
		t.Errorf("%q", a.str)
	}

	b := infoer{}
	webserv.LogInfo(&b, "b")
	webserv.LogInfo(&b, "\nb\n")
	if b.str != "b\nb" {
		t.Errorf("%q", b.str)
	}

	var c bytes.Buffer
	webserv.LogInfo(&c, "c")
	webserv.LogInfo(&c, "\nc\n")
	if c.String() != "c\n\nc\n" {
		t.Errorf("%q", c.String())
	}
}

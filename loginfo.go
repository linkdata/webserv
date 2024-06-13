package webserv

import (
	"fmt"
	"io"
	"strings"
)

type printlner interface {
	Println(v ...any)
}

type infoer interface {
	Info(msg string, args ...any)
}

func LogInfo(logger any, format string, args ...any) {
	msg := fmt.Sprintf(strings.TrimRight(format, "\n"), args...)
	switch x := logger.(type) {
	case printlner:
		x.Println(msg)
	case infoer:
		x.Info(msg)
	case io.Writer:
		_, _ = fmt.Fprintln(x, msg)
	}
}

package webserv

import (
	"fmt"
	"io"
)

type printfer interface {
	Printf(format string, v ...any)
}

type infoer interface {
	Info(msg string, args ...any)
}

func LogInfo(logger any, msg string, args ...any) {
	switch x := logger.(type) {
	case infoer:
		x.Info(msg, args...)
	case printfer:
		x.Printf(msg, args...)
	case io.Writer:
		_, _ = fmt.Fprintf(x, msg, args...)
	}
}

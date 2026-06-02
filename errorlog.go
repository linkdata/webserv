package webserv

import (
	"bytes"
	"log"
	"net/http"
)

const tlsHandshakeErrorLogPrefix = "http: TLS handshake error from "

var tlsHandshakeErrorLogPrefixBytes = []byte(tlsHandshakeErrorLogPrefix)

type tlsErrorLogWriter struct {
	previous *log.Logger
}

func (w tlsErrorLogWriter) Write(p []byte) (n int, err error) {
	if !bytes.HasPrefix(p, tlsHandshakeErrorLogPrefixBytes) {
		// Preserve file attribution for loggers using Lshortfile or Llongfile.
		if w.previous != nil {
			err = w.previous.Output(4, string(p))
		} else {
			err = log.Output(4, string(p))
		}
	}
	if err == nil {
		n = len(p)
	}
	return
}

// installTLSErrorLogFilter wraps srv.ErrorLog so that "http: TLS handshake
// error" lines are dropped while all other output is forwarded to the previous
// [net/http.Server.ErrorLog] (or the standard logger if it was nil).
//
// It performs a single write to srv.ErrorLog and must be called before
// [net/http.Server.Serve] starts. Doing so orders the write before every
// connection goroutine that later reads srv.ErrorLog, so the filter is never
// mutated again while connections are being served or drained.
func installTLSErrorLogFilter(srv *http.Server) {
	srv.ErrorLog = log.New(tlsErrorLogWriter{previous: srv.ErrorLog}, "", 0)
}

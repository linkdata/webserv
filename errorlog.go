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

func filterTLSErrorLog(srv *http.Server) (restore func()) {
	previous := srv.ErrorLog
	srv.ErrorLog = log.New(tlsErrorLogWriter{previous: previous}, "", 0)
	return func() {
		srv.ErrorLog = previous
	}
}

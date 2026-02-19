[![build](https://github.com/linkdata/webserv/actions/workflows/go.yml/badge.svg)](https://github.com/linkdata/webserv/actions/workflows/go.yml)
[![coverage](https://github.com/linkdata/webserv/blob/coverage/main/badge.svg)](https://html-preview.github.io/?url=https://github.com/linkdata/webserv/blob/coverage/main/report.html)
[![goreport](https://goreportcard.com/badge/github.com/linkdata/webserv)](https://goreportcard.com/report/github.com/linkdata/webserv)
[![Docs](https://godoc.org/github.com/linkdata/webserv?status.svg)](https://godoc.org/github.com/linkdata/webserv)

# webserv

Thin web service library.

Given a listen address, certificate directory, user name and data directory:

* If certificate directory is not blank, reads `fullchain.pem` and `privkey.pem` from it.
* If the listen address does not specify a port, default port depends on initial user privileges and if we have a certificate.
* Starts listening on the address and port.
* If listening succeeds but a later setup step fails, `Listen()` still returns an error and closes the listener, but `cfg.ListenURL` may already have been populated.
* If user name is given, switch to that user.
* If data directory is given, create it if needed.
* When serving, listen for SIGINT and SIGTERM and do a controlled shutdown.

## Usage

`go get github.com/linkdata/webserv`

```go
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/linkdata/webserv"
)

var (
	flagAddress   = flag.String("address", os.Getenv("WEBSERV_ADDRESS"), "serve HTTP requests on given [address][:port]")
	flagCertDir   = flag.String("certdir", os.Getenv("WEBSERV_CERTDIR"), "where to find fullchain.pem and privkey.pem")
	flagUser      = flag.String("user", envOrDefault("WEBSERV_USER", "www-data"), "switch to this user after startup (*nix only)")
	flagDataDir   = flag.String("datadir", envOrDefault("WEBSERV_DATADIR", "$HOME"), "where to store data files after startup")
	flagListenURL = flag.String("listenurl", os.Getenv("WEBSERV_LISTENURL"), "specify the external URL clients can reach us at")
)

func envOrDefault(envvar, defval string) (s string) {
	if s = os.Getenv(envvar); s == "" {
		s = defval
	}
	return
}

func main() {
	flag.Parse()

	cfg := webserv.Config{
		Address:   *flagAddress,
		CertDir:   *flagCertDir,
		User:      *flagUser,
		DataDir:   *flagDataDir,
		ListenURL: *flagListenURL,
		Logger:    slog.Default(),
	}

	http.DefaultServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html><body>Hello world!</body></html>"))
	})

	l, err := cfg.Listen()
	if err == nil {
		if err = cfg.Serve(context.Background(), l, nil); err == nil {
			return
		}
	}
	slog.Error(err.Error())
}
```

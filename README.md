[![build](https://github.com/linkdata/webserv/actions/workflows/go.yml/badge.svg)](https://github.com/linkdata/webserv/actions/workflows/go.yml)
[![coverage](https://coveralls.io/repos/github/linkdata/webserv/badge.svg?branch=main)](https://coveralls.io/github/linkdata/webserv?branch=main)
[![goreport](https://goreportcard.com/badge/github.com/linkdata/webserv)](https://goreportcard.com/report/github.com/linkdata/webserv)
[![Docs](https://godoc.org/github.com/linkdata/webserv?status.svg)](https://godoc.org/github.com/linkdata/webserv)

# webserv

Thin web server stub.

Given a listen address, certificate directory, user name and data directory:

* If certificate directory is not blank, reads `fullchain.pem` and `privkey.pem` from it.
* If the listen address does not specify a port, default port depends on initial user privileges and if we have a certificate.
* Starts listening on the address and port.
* If user name is given, switch to that user.
* If data directory is given, create it if needed.

## Usage

`go get github.com/linkdata/webserv`

```go
package main

import (
	"flag"
	"log/slog"
	"net/http"

	"github.com/linkdata/webserv"
)

var (
	flagListen  = flag.String("listen", "", "serve HTTP requests on given [address][:port]")
	flagCertDir = flag.String("certdir", "", "where to find fullchain.pem and privkey.pem")
	flagUser    = flag.String("user", "www-data", "switch to this user after startup (*nix only)")
	flagDataDir = flag.String("datadir", "$HOME", "where to store data files after startup")
)

func main() {
	flag.Parse()

	cfg := webserv.Config{
		Address: *flagListen,
		CertDir: *flagCertDir,
		User:    *flagUser,
		DataDir: *flagDataDir,
		Logger:  slog.Default(),
	}

	http.DefaultServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html><body>Hello world!</body></html>"))
	})

	l, err := cfg.Listen()
	if err == nil {
		err = cfg.Serve(context.Background(), l, nil)
	}
	slog.Error(err.Error())
}
```
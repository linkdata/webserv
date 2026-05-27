[![build](https://github.com/linkdata/webserv/actions/workflows/go.yml/badge.svg)](https://github.com/linkdata/webserv/actions/workflows/go.yml)
[![coverage](https://github.com/linkdata/webserv/blob/coverage/main/badge.svg)](https://html-preview.github.io/?url=https://github.com/linkdata/webserv/blob/coverage/main/report.html)
[![goreport](https://goreportcard.com/badge/github.com/linkdata/webserv)](https://goreportcard.com/report/github.com/linkdata/webserv)
[![Docs](https://godoc.org/github.com/linkdata/webserv?status.svg)](https://godoc.org/github.com/linkdata/webserv)

# webserv

Thin web service library.

Given a listen address, certificate directory, user name and data directory:

* If certificate directory is not blank, reads `fullchain.pem` and `privkey.pem` from it.
* If the listen address does not specify a port, default port depends on initial user privileges and if we have a certificate. To specify only a port, use `:port`.
* Starts listening on the address and port.
* If listening succeeds but a later setup step fails, `Listen()` still returns an error and closes the listener, but `cfg.ListenURL` may already have been populated.
* If user name is given, switch to that user.
* If data directory is given, create it if needed.
* When serving, listen for SIGINT and SIGTERM and do a controlled shutdown.
* `ServeWith` requires non-nil `ctx`, `srv`, and `listener`; panics from `srv.Serve` are recovered and returned as an error matching `ErrServePanic`.
* Path values are treated as trusted config: certificate filenames and data-dir suffixes may use `..` and symlinks and can resolve outside their base directories.

## Why use this instead of net/http directly?

Wiring up `http.Server` and `net.Listener` by hand is easy to get subtly wrong. This package bundles the safe defaults and lifecycle handling you would otherwise have to remember to add yourself.

### Security

* **Drops privileges safely (Unix only).** Bind to a privileged port (80/443) as root, then switch to an unprivileged `User`. Supplementary groups, GID and UID are dropped in the correct order (`setgroups` → `setgid` → `setuid`), and `HOME`/`USER`/`XDG_CONFIG_HOME` are fixed up to match.
* **Sane timeouts by default.** `Serve` sets `ReadHeaderTimeout` and `IdleTimeout`. A bare `http.Server{}` has no timeouts at all, leaving it open to Slowloris-style connection exhaustion.
* **TLS 1.3 minimum.** When a certificate is loaded, the listener pins `MinVersion` to TLS 1.3 instead of relying on the standard library default.
* **Quiet TLS handshake errors.** Failed handshakes (port scanners, plain HTTP sent to an HTTPS port) no longer flood your logs by default; set `LogTLSErrors` to keep them.
* **Recovers serve panics.** A panic inside `srv.Serve` is recovered and returned as an error matching `ErrServePanic` instead of taking down the process.

### Convenience

* **One call does the setup.** `ListenAndServe` loads certificates, opens the listener, drops privileges, prepares the data directory and serves — in the right order, with errors propagated.
* **Automatic address defaults.** Port and scheme are chosen from privilege level and whether a certificate was loaded (80/443 as root, 8080/8443 otherwise). Override with a full address or just `:port`.
* **Graceful shutdown.** SIGINT/SIGTERM, or canceling the context, triggers `srv.Shutdown` bounded by `ShutdownTimeLimit`, so in-flight requests can finish and the port is released cleanly.
* **A connectable URL.** `cfg.ListenURL` is filled in with a printable, reachable URL (resolving wildcard/loopback binds to `localhost` or the certificate's DNS name) — handy for logs and links.
* **Managed data directory.** Resolves `DataDir` to an absolute path and optionally creates it, defaulting under the user config directory.
* **Bring your own logger.** The `Logger` interface matches `log/slog`, so structured startup and shutdown logging drops right in.

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

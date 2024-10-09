package webserv_test

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

func Example() {
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

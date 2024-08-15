package webserv_test

import (
	"context"
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

func Example() {
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

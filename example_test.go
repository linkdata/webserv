package webserv_test

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"syscall"
	"time"

	"github.com/linkdata/webserv"
)

var (
	flagListen  = flag.String("listen", "", "serve HTTP requests on given [address][:port]")
	flagCertDir = flag.String("certdir", "", "where to find fullchain.pem and privkey.pem")
	flagUser    = flag.String("user", "www-data", "switch to this user after startup (*nix only)")
	flagDataDir = flag.String("datadir", "$HOME", "where to store data files after startup")
)

// make sure we don't time out on the Go playground
func dontTimeOutOnGoPlayground() {
	go func() {
		time.Sleep(time.Second)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()
}

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
		dontTimeOutOnGoPlayground()
		err = cfg.Serve(context.Background(), l, nil)
	}
	slog.Error(err.Error())
}

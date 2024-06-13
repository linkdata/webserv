package webserv_test

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
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
		slog.Info("goodbye!")
		os.Exit(0)
	}()
}

func Example() {
	flag.Parse()

	cfg := webserv.Config{
		Listen:  *flagListen,
		CertDir: *flagCertDir,
		User:    *flagUser,
		DataDir: *flagDataDir,
	}

	l, err := cfg.Apply(slog.Default())
	if err == nil {
		defer l.Close()
		slog.Info("listening", "address", l.Addr(), "url", cfg.ListenURL)
		http.DefaultServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("<html><body>Hello world!</body></html>"))
		})
		dontTimeOutOnGoPlayground()
		err = http.Serve(l, nil)
	}
	slog.Error(err.Error())
}

package webserv_test

import (
	"flag"
	"log"
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

func Example() {
	flag.Parse()

	cfg := webserv.Config{
		Listen:  *flagListen,
		CertDir: *flagCertDir,
		User:    *flagUser,
		DataDir: *flagDataDir,
	}

	l, err := cfg.Apply(log.Default())
	if err == nil {
		defer l.Close()
		log.Printf("listening on %q", cfg.ListenURL)
		http.DefaultServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("<html><body>Hello world!</body></html>"))
		})
		go func() {
			time.Sleep(time.Second)
			log.Printf("goodbye!")
			os.Exit(0)
		}()
		err = http.Serve(l, nil)
	}
	log.Fatal(err)
}

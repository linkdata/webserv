package webserv_test

import (
	"flag"
	"log"

	"github.com/linkdata/webserv"
)

var (
	flagListen  = flag.String("listen", "", "serve HTTP requests on given [address][:port]")
	flagCertDir = flag.String("certdir", "", "where to find fullchain.pem and privkey.pem")
	flagUser    = flag.String("user", "", "switch to this user after startup (*nix only)")
	flagDataDir = flag.String("datadir", "", "where to store data files after startup")
)

func Example() {
	flag.Parse()

	cfg := webserv.Config{
		Listen:               *flagListen,
		CertDir:              *flagCertDir,
		User:                 *flagUser,
		DataDir:              *flagDataDir,
		DefaultDataDirSuffix: "webserv_example",
	}

	if l, err := cfg.Apply(log.Default()); err == nil {
		defer l.Close()
		log.Print("listening on", cfg.ListenURL)
	} else {
		log.Fatal(err)
	}
}

package webserv

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
)

const (
	FullchainPem = "fullchain.pem"
	PrivkeyPem   = "privkey.pem"
)

// Listener creates a net.Listener given an optional preferred address or port
// and an optional directory containing certificate files.
//
// If certDir is not empty, it calls LoadCert to load fullchain.pem and privkey.pem.
//
// The listener will default to all addresses and standard port
// depending on privileges and if a certificate was loaded or not.
//
// These defaults can be overridden with the listenAddr argument.
//
// Returns the net.Listener and listenURL if there was no error.
// If certificates were successfully loaded, absCertDir will be the absolute path to that directory.
func Listener(listenAddr, certDir string) (l net.Listener, listenUrl, absCertDir string, err error) {
	var cert *tls.Certificate
	if cert, absCertDir, err = LoadCert(certDir); err == nil {
		var schemesuffix string
		if cert != nil {
			schemesuffix = "s"
			l, err = tls.Listen("tcp", defaultAddress(listenAddr, ":443", ":8443"),
				&tls.Config{
					Certificates: []tls.Certificate{*cert},
					MinVersion:   tls.VersionTLS13,
				},
			)
		} else {
			l, err = net.Listen("tcp", defaultAddress(listenAddr, ":80", ":8080"))
		}
		if l != nil {
			listenUrl = fmt.Sprintf("http%s://%s", schemesuffix, listenAddrString(l))
		}
	}
	return
}

func defaultAddress(address, defaultpriv, defaultother string) string {
	if address == "" {
		address = defaultpriv
		if os.Geteuid() > 0 {
			address = defaultother
		}
	}
	return address
}

func listenAddrString(l net.Listener) (addr string) {
	addr = l.Addr().String()
	if host, port, err := net.SplitHostPort(addr); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			if ip.IsUnspecified() || ip.IsLoopback() {
				addr = net.JoinHostPort("localhost", port)
			}
		}
	}
	return
}

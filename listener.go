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

type Listener struct {
	net.Listener
	CertDir string
}

func (l Listener) ListenURL() string {
	addr := l.Addr().String()
	if host, port, err := net.SplitHostPort(addr); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			if ip.IsUnspecified() || ip.IsLoopback() {
				addr = net.JoinHostPort("localhost", port)
			}
		}
	}
	schemesuffix := "s"
	if l.CertDir == "" {
		schemesuffix = ""
	}
	return fmt.Sprintf("http%s://%s", schemesuffix, addr)
}

func NewListener(wantAddress, certDir string) (l Listener, err error) {
	var cert *tls.Certificate
	if cert, l.CertDir, err = LoadCert(certDir); err == nil {
		if cert != nil {
			l.Listener, err = tls.Listen("tcp", defaultAddress(wantAddress, ":443", ":8443"),
				&tls.Config{
					Certificates: []tls.Certificate{*cert},
					MinVersion:   tls.VersionTLS12,
				},
			)
		} else {
			l.Listener, err = net.Listen("tcp", defaultAddress(wantAddress, ":80", ":8080"))
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

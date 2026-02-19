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
func Listener(listenAddr, certDir, fullchainPem, privkeyPem, overrideUrl string) (l net.Listener, listenUrl, absCertDir string, err error) {
	var cert *tls.Certificate
	if cert, absCertDir, err = LoadCert(certDir, fullchainPem, privkeyPem); err == nil {
		var schemesuffix string
		if cert != nil {
			schemesuffix = "s"
			l, err = tls.Listen("tcp", defaultAddress(listenAddr, "443", "8443"),
				&tls.Config{
					Certificates: []tls.Certificate{*cert},
					MinVersion:   tls.VersionTLS13,
				},
			)
		} else {
			l, err = net.Listen("tcp", defaultAddress(listenAddr, "80", "8080"))
		}
		if l != nil {
			if listenUrl = overrideUrl; listenUrl == "" {
				listenUrl = fmt.Sprintf("http%s://%s", schemesuffix, listenUrlString(l, cert))
			}
		}
	}
	return
}

func defaultAddress(address, defaultpriv, defaultother string) (result string) {
	result = address
	if _, _, err := net.SplitHostPort(address); err != nil {
		defaultPort := defaultpriv
		if os.Geteuid() > 0 {
			defaultPort = defaultother
		}
		result = net.JoinHostPort(address, defaultPort)
	}
	return
}

func localhostOrDNSName(cert *tls.Certificate) string {
	if cert != nil && cert.Leaf != nil && len(cert.Leaf.DNSNames) > 0 {
		name := cert.Leaf.DNSNames[0]
		if host, _, err := net.SplitHostPort(name); err == nil {
			name = host
		}
		return name
	}
	return "localhost"
}

func listenUrlString(l net.Listener, cert *tls.Certificate) (addr string) {
	addr = l.Addr().String()
	if host, port, err := net.SplitHostPort(addr); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			if ip.IsUnspecified() || ip.IsLoopback() {
				addr = net.JoinHostPort(localhostOrDNSName(cert), port)
			}
		}
	}
	return
}

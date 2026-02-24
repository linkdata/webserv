package webserv

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strings"
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
		var bindAddr string
		var schemesuffix string
		if cert != nil {
			schemesuffix = "s"
			if bindAddr, err = normalizeListenAddr(listenAddr, "443", "8443"); err == nil {
				l, err = tls.Listen("tcp", bindAddr,
					&tls.Config{
						Certificates: []tls.Certificate{*cert},
						MinVersion:   tls.VersionTLS13,
					},
				)
			}
		} else {
			if bindAddr, err = normalizeListenAddr(listenAddr, "80", "8080"); err == nil {
				l, err = net.Listen("tcp", bindAddr)
			}
		}
		if l != nil {
			if listenUrl = overrideUrl; listenUrl == "" {
				listenUrl = fmt.Sprintf("http%s://%s", schemesuffix, listenUrlString(l, cert))
			}
		}
	}
	return
}

func normalizeListenAddr(address, defaultpriv, defaultother string) (result string, err error) {
	if _, _, err = net.SplitHostPort(address); err == nil {
		// host and port both present
		if strings.HasPrefix(address, "[]") {
			err = net.InvalidAddrError(address)
		} else {
			result = address
		}
		return
	}

	err = nil
	defaultPort := defaultpriv
	if os.Geteuid() > 0 {
		defaultPort = defaultother
	}

	result = address
	if strings.HasPrefix(address, "[") {
		if len(address) > 2 && strings.HasSuffix(address, "]") {
			result = address[1 : len(address)-1]
		} else {
			result = ""
			err = net.InvalidAddrError(address)
		}
	}
	if err == nil {
		result = net.JoinHostPort(result, defaultPort)
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

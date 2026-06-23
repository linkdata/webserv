package webserv

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
)

const (
	// FullchainPem is the default certificate chain filename used by LoadCert.
	FullchainPem = "fullchain.pem"
	// PrivkeyPem is the default private key filename used by LoadCert.
	PrivkeyPem = "privkey.pem"
)

// Listener creates a [net.Listener] given an optional preferred address
// and an optional directory containing certificate files.
//
// If certDir is not empty, it calls [LoadCert] to load fullchain.pem and privkey.pem.
//
// The listener will default to all addresses and standard port
// depending on privileges and if a certificate was loaded or not.
//
// These defaults can be overridden with the listenAddr argument.
// To specify only a port, use an address like ":8080".
//
// Returns the [net.Listener] and listenURL if there was no error.
// absCertDir is the resolved absolute path to certDir whenever certDir was
// non-empty and could be resolved, even if loading the certificate then failed.
func Listener(listenAddr, certDir, fullchainPem, privkeyPem, overrideUrl string) (l net.Listener, listenUrl, absCertDir string, err error) {
	var cert *tls.Certificate
	if cert, absCertDir, err = LoadCert(certDir, fullchainPem, privkeyPem); err == nil {
		var bindAddr string
		var schemesuffix string
		if cert != nil {
			schemesuffix = "s"
			if bindAddr, err = normalizeListenAddr(listenAddr, "443", "8443"); err == nil {
				l, err = tls.Listen(
					"tcp", bindAddr,
					&tls.Config{
						Certificates: []tls.Certificate{*cert},
						MinVersion:   tls.VersionTLS13,
						NextProtos:   []string{"h2", "http/1.1"},
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

func normalizeListenAddr(address, defaultpriv, defaultother string) (string, error) {
	// A complete "host:port" (including "[host]:port" and ":port") is kept as-is.
	// The empty bracketed host "[]" is the one exception: unlike a port-only
	// ":port" it is never a valid host.
	if _, _, err := net.SplitHostPort(address); err == nil {
		if strings.HasPrefix(address, "[]") {
			return "", net.InvalidAddrError(address)
		}
		return address, nil
	}

	// No port: address is a bare host. A leading "[" must wrap a valid IP
	// literal so net.JoinHostPort can re-bracket it below; otherwise reject it
	// rather than turning it into a bogus bind address.
	host := address
	if inner, ok := strings.CutPrefix(host, "["); ok {
		lit, closed := strings.CutSuffix(inner, "]")
		if !closed {
			return "", net.InvalidAddrError(address)
		}
		if _, err := netip.ParseAddr(lit); err != nil {
			return "", net.InvalidAddrError(address)
		}
		host = lit
	}
	return net.JoinHostPort(host, defaultListenPort(os.Geteuid(), defaultpriv, defaultother)), nil
}

func defaultListenPort(euid int, defaultpriv, defaultother string) (port string) {
	port = defaultother
	if euid == 0 {
		port = defaultpriv
	}
	return
}

func localhostOrDNSName(cert *tls.Certificate) string {
	if cert != nil && cert.Leaf != nil && len(cert.Leaf.DNSNames) > 0 {
		name := cert.Leaf.DNSNames[0]
		if host, _, err := net.SplitHostPort(name); err == nil {
			name = host
		}
		// Guard against a certificate whose first DNS name is empty (or strips
		// to empty): an empty host would build an unconnectable ":port" URL.
		if name != "" {
			return name
		}
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

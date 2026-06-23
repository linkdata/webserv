package webserv

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"os"
	"testing"
)

// certWithDNSNames builds a *tls.Certificate carrying only a synthetic leaf
// with the given SubjectAltName DNS names. localhostOrDNSName and
// listenUrlString read only cert.Leaf.DNSNames, so no real key material is
// needed.
func certWithDNSNames(names ...string) *tls.Certificate {
	return &tls.Certificate{Leaf: &x509.Certificate{DNSNames: names}}
}

func TestNormalizeListenAddr_HostWithoutPortGetsDefaultPort(t *testing.T) {
	httpDefault := "80"
	httpsDefault := "443"
	if os.Geteuid() != 0 {
		httpDefault = "8080"
		httpsDefault = "8443"
	}

	if got, err := normalizeListenAddr("", "80", "8080"); got != ":"+httpDefault || err != nil {
		t.Fatalf("normalizeListenAddr(\"\") = (%q, %v), want (%q, nil)", got, err, ":"+httpDefault)
	}
	if got, err := normalizeListenAddr("localhost", "80", "8080"); err != nil || got != "localhost:"+httpDefault {
		t.Fatalf("normalizeListenAddr(\"localhost\") = (%q, %v), want (%q, nil)", got, err, "localhost:"+httpDefault)
	}
	if got, err := normalizeListenAddr("localhost", "443", "8443"); err != nil || got != "localhost:"+httpsDefault {
		t.Fatalf("normalizeListenAddr(\"localhost\") = (%q, %v), want (%q, nil)", got, err, "localhost:"+httpsDefault)
	}
}

func TestNormalizeListenAddr_BracketedIPv6WithoutPort(t *testing.T) {
	httpDefault := "80"
	if os.Geteuid() != 0 {
		httpDefault = "8080"
	}

	if got, err := normalizeListenAddr("[::1]", "80", "8080"); err != nil || got != "[::1]:"+httpDefault {
		t.Fatalf("normalizeListenAddr(\"[::1]\") = (%q, %v), want (%q, nil)", got, err, "[::1]:"+httpDefault)
	}
}

func TestNormalizeListenAddr_MalformedBracketHostRejected(t *testing.T) {
	// Without a port, brackets must wrap a valid IP literal. Besides empty or
	// unterminated cases, reject bracketed non-literals and surplus brackets.
	for _, in := range []string{"[]", "[]:", "[]:0", "[::1", "[]]", "[[::1]]", "[localhost]"} {
		got, err := normalizeListenAddr(in, "80", "8080")
		if err == nil {
			t.Fatalf("normalizeListenAddr(%q) = (%q, nil), want invalid address error", in, got)
		}
		var invalid net.InvalidAddrError
		if !errors.As(err, &invalid) {
			t.Fatalf("normalizeListenAddr(%q) error = %T (%v), want %T", in, err, err, invalid)
		}
	}
}

func TestNormalizeListenAddr_BracketedZonedIPv6Accepted(t *testing.T) {
	// A bracketed link-local literal with a zone is valid; netip.ParseAddr
	// accepts the zone where net.ParseIP would not.
	httpDefault := "80"
	if os.Geteuid() != 0 {
		httpDefault = "8080"
	}
	const in = "[fe80::1%eth0]"
	if got, err := normalizeListenAddr(in, "80", "8080"); err != nil || got != in+":"+httpDefault {
		t.Fatalf("normalizeListenAddr(%q) = (%q, %v), want (%q, nil)", in, got, err, in+":"+httpDefault)
	}
}

func TestNormalizeListenAddr_MalformedBracketResultIsEmpty(t *testing.T) {
	// A malformed bracket address must return an empty result alongside the
	// error, never a usable address such as ":8080".
	for _, in := range []string{"[]", "[::1", "[]]", "[[::1]]"} {
		got, err := normalizeListenAddr(in, "80", "8080")
		if err == nil {
			t.Fatalf("normalizeListenAddr(%q) = (%q, nil), want error", in, got)
		}
		if got != "" {
			t.Fatalf("normalizeListenAddr(%q) result = %q on error, want empty string", in, got)
		}
	}
}

func TestDefaultListenPort_RootUsesPrivilegedDefault(t *testing.T) {
	if got := defaultListenPort(0, "80", "8080"); got != "80" {
		t.Fatalf("defaultListenPort(0) = %q, want %q", got, "80")
	}
}

func TestDefaultListenPort_NonRootUsesOtherDefault(t *testing.T) {
	if got := defaultListenPort(1000, "80", "8080"); got != "8080" {
		t.Fatalf("defaultListenPort(1000) = %q, want %q", got, "8080")
	}
}

func TestDefaultListenPort_NegativeEUIDUsesOtherDefault(t *testing.T) {
	// A negative euid (such as the -1 os.Geteuid returns on Windows) is not
	// root and must use the unprivileged default port.
	if got := defaultListenPort(-1, "80", "8080"); got != "8080" {
		t.Fatalf("defaultListenPort(-1) = %q, want %q", got, "8080")
	}
}

func TestLocalhostOrDNSName(t *testing.T) {
	for _, tc := range []struct {
		name string
		cert *tls.Certificate
		want string
	}{
		{name: "nil cert", cert: nil, want: "localhost"},
		{name: "nil leaf", cert: &tls.Certificate{}, want: "localhost"},
		{name: "no DNS names", cert: certWithDNSNames(), want: "localhost"},
		{name: "DNS name", cert: certWithDNSNames("example.test"), want: "example.test"},
		{name: "DNS name with port", cert: certWithDNSNames("example.test:443"), want: "example.test"},
		// An empty first SAN must fall back to localhost, never "" (which would
		// build an unconnectable ":port" URL). Only the first name is consulted.
		{name: "empty first DNS name", cert: certWithDNSNames(""), want: "localhost"},
		{name: "empty before non-empty", cert: certWithDNSNames("", "example.test"), want: "localhost"},
		{name: "wildcard DNS name", cert: certWithDNSNames("*.example.test"), want: "localhost"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := localhostOrDNSName(tc.cert); got != tc.want {
				t.Fatalf("localhostOrDNSName() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestListenUrlString_UnspecifiedBindUsesCertDNSName(t *testing.T) {
	// Binding to an unspecified address must rewrite the host of the printable
	// URL to the certificate's DNS name (distinct from the localhost fallback).
	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	if got, want := listenUrlString(l, certWithDNSNames("example.test")), net.JoinHostPort("example.test", port); got != want {
		t.Fatalf("listenUrlString() = %q, want %q", got, want)
	}
	if got, want := listenUrlString(l, nil), net.JoinHostPort("localhost", port); got != want {
		t.Fatalf("listenUrlString() without cert = %q, want %q", got, want)
	}
}

func TestListenUrlString_UnspecifiedBindSkipsWildcardCertDNSName(t *testing.T) {
	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	if got, want := listenUrlString(l, certWithDNSNames("*.example.test")), net.JoinHostPort("localhost", port); got != want {
		t.Fatalf("listenUrlString() = %q, want %q", got, want)
	}
}

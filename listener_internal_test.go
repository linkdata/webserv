package webserv

import (
	"errors"
	"net"
	"os"
	"testing"
)

func TestNormalizeListenAddr_HostWithoutPortGetsDefaultPort(t *testing.T) {
	httpDefault := "80"
	httpsDefault := "443"
	if os.Geteuid() > 0 {
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
	if os.Geteuid() > 0 {
		httpDefault = "8080"
	}

	if got, err := normalizeListenAddr("[::1]", "80", "8080"); err != nil || got != "[::1]:"+httpDefault {
		t.Fatalf("normalizeListenAddr(\"[::1]\") = (%q, %v), want (%q, nil)", got, err, "[::1]:"+httpDefault)
	}
}

func TestNormalizeListenAddr_MalformedBracketHostRejected(t *testing.T) {
	for _, in := range []string{"[]", "[]:", "[]:0", "[::1"} {
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

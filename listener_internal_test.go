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

func TestNormalizeListenAddr_MalformedBracketResultIsEmpty(t *testing.T) {
	// A malformed bracket address must return an empty result alongside the
	// error, never a usable address such as ":8080".
	for _, in := range []string{"[]", "[::1"} {
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

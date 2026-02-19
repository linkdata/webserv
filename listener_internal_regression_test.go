package webserv

import (
	"os"
	"testing"
)

func TestDefaultAddress_HostWithoutPortGetsDefaultPort(t *testing.T) {
	httpDefault := "80"
	httpsDefault := "443"
	if os.Geteuid() > 0 {
		httpDefault = "8080"
		httpsDefault = "8443"
	}

	if got, want := defaultAddress("", "80", "8080"), ":"+httpDefault; got != want {
		t.Fatalf("http defaultAddress() = %q, want %q", got, want)
	}
	if got, want := defaultAddress("localhost", "80", "8080"), "localhost:"+httpDefault; got != want {
		t.Fatalf("http defaultAddress() = %q, want %q", got, want)
	}
	if got, want := defaultAddress("localhost", "443", "8443"), "localhost:"+httpsDefault; got != want {
		t.Fatalf("https defaultAddress() = %q, want %q", got, want)
	}
}

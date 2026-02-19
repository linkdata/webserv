package webserv_test

import (
	"net"
	"os/user"
	"testing"

	"github.com/linkdata/webserv"
)

func TestConfigListen_ErrorDoesNotReturnListener(t *testing.T) {
	// Pick a port that should be free for this test run.
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := probe.Addr().String()
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}

	const noSuchUser = "webserv-no-such-user-regression-test"
	if _, err := user.Lookup(noSuchUser); err == nil {
		t.Skipf("test user unexpectedly exists: %q", noSuchUser)
	}

	cfg := &webserv.Config{
		Address: addr,
		User:    noSuchUser,
	}
	l, err := cfg.Listen()
	if err == nil {
		if l != nil {
			_ = l.Close()
		}
		t.Fatal("expected Listen error")
	}
	if l != nil {
		defer func() { _ = l.Close() }()
		t.Fatalf("expected nil listener on Listen error, got %s", l.Addr().String())
	}
}

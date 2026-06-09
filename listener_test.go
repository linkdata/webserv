package webserv_test

import (
	"crypto/tls"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/linkdata/webserv"
)

func urlHostAndPort(t *testing.T, scheme string, l net.Listener) string {
	t.Helper()
	host, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		host = "localhost"
	}
	return scheme + "://" + net.JoinHostPort(host, port)
}

func withCertFiles(t *testing.T, fn func(destdir string)) {
	t.Helper()
	certPem := []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`)
	keyPem := []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`)
	destdir, err := os.MkdirTemp("", "weblistener")
	if err == nil {
		defer func() {
			if err := os.RemoveAll(destdir); err != nil {
				t.Error(err)
			}
		}()
		if err = os.WriteFile(filepath.Join(destdir, webserv.FullchainPem), certPem, 0o640); err == nil {
			if err = os.WriteFile(filepath.Join(destdir, webserv.PrivkeyPem), keyPem, 0o640); err == nil {
				fn(destdir)
			}
		}
	}
	if err != nil {
		t.Error(err)
	}
}

func TestRandomPort(t *testing.T) {
	gotListener, gotUrl, _, err := webserv.Listener("localhost:", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if gotListener == nil {
		t.Fatal("no listener")
	}
	_, p, err := net.SplitHostPort(gotListener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	if len(p) < 2 {
		t.Error(p)
	}
	want := "http://localhost:" + p
	if gotUrl != want {
		t.Errorf("want %q got %q", want, gotUrl)
	}
}

func TestNew(t *testing.T) {
	withCertFiles(t, func(destdir string) {
		type args struct {
			address string
			certDir string
			scheme  string
		}
		tests := []struct {
			name    string
			args    args
			wantErr bool
		}{
			{
				name: "http random port",
				args: args{
					address: "127.0.0.1:0",
					scheme:  "http",
				},
				wantErr: false,
			},
			{
				name: "http explicit random port",
				args: args{
					address: "localhost:0",
					scheme:  "http",
				},
			},
			{
				name: "https random port",
				args: args{
					address: "127.0.0.1:0",
					certDir: destdir,
					scheme:  "https",
				},
			},
			{
				name: "https explicit random port",
				args: args{
					address: "localhost:0",
					certDir: destdir,
					scheme:  "https",
				},
			},
			{
				name: "invalid port",
				args: args{
					address: "127.0.0.1:99999",
				},
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				gotListener, gotUrl, gotCertDir, err := webserv.Listener(tt.args.address, tt.args.certDir, "", "", "")
				if (err != nil) != tt.wantErr {
					t.Errorf("Listener() %q error = %v, wantErr %v", tt.args.address, err, tt.wantErr)
					return
				}
				if gotListener != nil {
					if want := urlHostAndPort(t, tt.args.scheme, gotListener); gotUrl != want {
						t.Errorf("ListenURL() = %v, want %v", gotUrl, want)
					}
					if gotCertDir != tt.args.certDir {
						t.Error(gotCertDir)
					}
					if err := gotListener.Close(); err != nil {
						t.Error(err)
					}
				}
			})
		}
	})
}

func TestListener_MalformedBracketHostRejected(t *testing.T) {
	for _, listenAddr := range []string{"[]", "[]:0", "[]:"} {
		l, _, _, err := webserv.Listener(listenAddr, "", "", "", "")
		addr := "<nil>"
		if l != nil {
			addr = l.Addr().String()
			_ = l.Close()
		}
		if err == nil {
			t.Fatalf("expected malformed listen address %q to fail, got listener on %q", listenAddr, addr)
		}
	}
}

func TestListener_TLSAdvertisesHTTP2(t *testing.T) {
	withCertFiles(t, func(destdir string) {
		l, _, _, err := webserv.Listener("127.0.0.1:0", destdir, "", "", "")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = l.Close() }()

		acceptDone := make(chan error, 1)
		go func() {
			var conn net.Conn
			var err error
			if conn, err = l.Accept(); err == nil {
				defer func() { _ = conn.Close() }()
				if err = conn.SetDeadline(time.Now().Add(time.Second)); err == nil {
					var tlsConn *tls.Conn
					var ok bool
					if tlsConn, ok = conn.(*tls.Conn); ok {
						err = tlsConn.Handshake()
					} else {
						err = errors.New("accepted connection is not TLS")
					}
				}
			}
			acceptDone <- err
		}()

		dialer := &net.Dialer{
			Timeout: time.Second,
		}
		conn, err := tls.DialWithDialer(dialer, "tcp", l.Addr().String(), &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"h2", "http/1.1"},
		})
		if err != nil {
			t.Fatal(err)
		}
		state := conn.ConnectionState()
		if err = conn.Close(); err != nil {
			t.Fatal(err)
		}
		if err = <-acceptDone; err != nil {
			t.Fatal(err)
		}
		if state.NegotiatedProtocol != "h2" {
			t.Fatalf("negotiated protocol = %q, want %q", state.NegotiatedProtocol, "h2")
		}
	})
}

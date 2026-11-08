package webserv_test

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/linkdata/webserv"
)

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
	defer func() { _ = gotListener.Close() }()
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

func TestListener_SpecificLoopbackURLUsesBoundIP(t *testing.T) {
	for _, host := range []string{"127.0.0.2", "::1", "::1%lo"} {
		t.Run(host, func(t *testing.T) {
			bindAddr := net.JoinHostPort(host, "0")
			probe, err := net.Listen("tcp", bindAddr)
			if err != nil {
				t.Skipf("cannot bind %s: %v", host, err)
			}
			_ = probe.Close()

			l, gotURL, _, err := webserv.Listener(bindAddr, "", "", "", "")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = l.Close() }()

			if want := (&url.URL{Scheme: "http", Host: l.Addr().String()}).String(); gotURL != want {
				t.Fatalf("Listener URL = %q, want %q", gotURL, want)
			}
		})
	}
}

func TestListener_MappedCanonicalLoopbackKeepsTLSName(t *testing.T) {
	const bindAddr = "[::ffff:127.0.0.1]:0"
	probe, err := net.Listen("tcp", bindAddr)
	if err != nil {
		t.Skipf("cannot bind %s: %v", bindAddr, err)
	}
	_ = probe.Close()

	withCertFiles(t, func(certDir string) {
		l, gotURL, _, err := webserv.Listener(bindAddr, certDir, "", "", "")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = l.Close() }()
		_, port, err := net.SplitHostPort(l.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		if want := "https://localhost:" + port; gotURL != want {
			t.Fatalf("Listener URL = %q, want %q", gotURL, want)
		}
	})
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
					wantHostPort := gotListener.Addr().String()
					if tt.args.address == "localhost:0" || tt.args.address == "127.0.0.1:0" {
						_, port, err := net.SplitHostPort(wantHostPort)
						if err != nil {
							t.Fatal(err)
						}
						wantHostPort = net.JoinHostPort("localhost", port)
					}
					if want := tt.args.scheme + "://" + wantHostPort; gotUrl != want {
						t.Errorf("Listener() URL = %q, want %q", gotUrl, want)
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

func TestListener_EmptyBracketHostWithPort(t *testing.T) {
	for _, listenAddr := range []string{"[]:0", "[]:"} {
		l, gotURL, _, err := webserv.Listener(listenAddr, "", "", "", "")
		if err != nil {
			t.Fatalf("Listener(%q): %v", listenAddr, err)
		}
		defer func() { _ = l.Close() }()

		host, port, err := net.SplitHostPort(l.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		if ip := net.ParseIP(host); ip == nil || !ip.IsUnspecified() || port == "0" {
			t.Fatalf("Listener(%q) bound %q, want wildcard address and assigned port", listenAddr, l.Addr())
		}
		if want := "http://localhost:" + port; gotURL != want {
			t.Fatalf("Listener(%q) URL = %q, want %q", listenAddr, gotURL, want)
		}
	}
}

func TestListener_BareEmptyBracketHostRejected(t *testing.T) {
	l, _, _, err := webserv.Listener("[]", "", "", "", "")
	if l != nil {
		_ = l.Close()
		t.Fatalf("Listener(\"[]\") returned listener on %q", l.Addr())
	}
	if err == nil {
		t.Fatal("Listener(\"[]\") succeeded, want invalid address error")
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

func TestConfigServe_TLSProtocolsFollowGODEBUGAtServe(t *testing.T) {
	for _, tc := range []struct {
		name           string
		listenGODEBUG  string
		serveGODEBUG   string
		wantProtoMajor int
	}{
		{name: "enable HTTP/2", listenGODEBUG: "http2server=0", serveGODEBUG: "http2server=1", wantProtoMajor: 2},
		{name: "disable HTTP/2", listenGODEBUG: "http2server=1", serveGODEBUG: "http2server=0", wantProtoMajor: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Listener does not read GODEBUG; the contrary value verifies that
			// the setting at serve time wins.
			t.Setenv("GODEBUG", tc.listenGODEBUG)

			withCertFiles(t, func(destdir string) {
				cfg := &webserv.Config{
					Address: "127.0.0.1:0",
					CertDir: destdir,
				}
				l, err := cfg.Listen()
				if err != nil {
					t.Fatal(err)
				}

				t.Setenv("GODEBUG", tc.serveGODEBUG)
				ctx, cancel := context.WithCancel(t.Context())
				serveDone := make(chan error, 1)
				go func() {
					serveDone <- cfg.Serve(ctx, l, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusNoContent)
					}))
				}()
				defer func() {
					cancel()
					if err := <-serveDone; !errors.Is(err, context.Canceled) {
						t.Errorf("Serve() error = %v, want %v", err, context.Canceled)
					}
				}()

				transport := &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
					ForceAttemptHTTP2: true,
				}
				defer transport.CloseIdleConnections()

				client := &http.Client{
					Transport: transport,
					Timeout:   3 * time.Second,
				}
				resp, err := client.Get("https://" + l.Addr().String())
				if err != nil {
					t.Fatal(err)
				}
				if err = resp.Body.Close(); err != nil {
					t.Fatal(err)
				}
				if resp.StatusCode != http.StatusNoContent {
					t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
				}
				if resp.ProtoMajor != tc.wantProtoMajor {
					t.Fatalf("protocol = %q, want major version %d", resp.Proto, tc.wantProtoMajor)
				}
			})
		})
	}
}

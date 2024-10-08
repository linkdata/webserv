package webserv_test

import (
	"os"
	"path"
	"testing"

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
		if err = os.WriteFile(path.Join(destdir, webserv.FullchainPem), certPem, 0640); err == nil {
			if err = os.WriteFile(path.Join(destdir, webserv.PrivkeyPem), keyPem, 0640); err == nil {
				fn(destdir)
			}
		}
	}
	if err != nil {
		t.Error(err)
	}
}

func TestNew(t *testing.T) {
	withCertFiles(t, func(destdir string) {
		httpPort := "8080"
		httpsPort := "8443"
		if os.Getuid() < 1 {
			httpPort = "80"
			httpsPort = "443"
		}
		type args struct {
			wantAddress string
			certDir     string
		}
		tests := []struct {
			name    string
			args    args
			wantErr bool
		}{
			{
				name:    "http://localhost:" + httpPort,
				args:    args{},
				wantErr: false,
			},
			{
				name: "http://localhost:8888",
				args: args{
					wantAddress: "localhost:8888",
				},
			},
			{
				name: "https://localhost:" + httpsPort,
				args: args{
					certDir: destdir,
				},
			},
			{
				name: "https://localhost:4443",
				args: args{
					wantAddress: "localhost:4443",
					certDir:     destdir,
				},
			},
			{
				name: "invalid port",
				args: args{
					wantAddress: "127.0.0.1:99999",
				},
				wantErr: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				gotListener, gotUrl, gotCertDir, err := webserv.Listener(tt.args.wantAddress, tt.args.certDir, "", "", "")
				if (err != nil) != tt.wantErr {
					t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if gotListener != nil {
					if gotUrl != tt.name {
						t.Errorf("ListenURL() = %v, want %v", gotUrl, tt.name)
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

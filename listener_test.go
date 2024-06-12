package webserv_test

import (
	"os"
	"testing"

	"github.com/linkdata/webserv"
)

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
				gotListener, gotUrl, gotCertDir, err := webserv.Listener(tt.args.wantAddress, tt.args.certDir)
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

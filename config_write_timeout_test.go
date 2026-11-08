package webserv

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"
)

func TestDefaultHTTPServer_HTTP2WriteTimeout(t *testing.T) {
	t.Setenv("GODEBUG", "http2server=1,http2client=1")
	synctest.Test(t, func(t *testing.T) {
		writeDone := make(chan error, 1)
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if err := http.NewResponseController(w).Flush(); err != nil {
				writeDone <- err
				return
			}
			_, err := w.Write(make([]byte, 64<<10))
			writeDone <- err
		})

		server := httptest.NewTestServer(t, handler)
		server.EnableHTTP2 = true
		server.Config = defaultHTTPServer(handler)
		client := server.Client()
		client.Transport.(*http.Transport).HTTP2 = &http.HTTP2Config{MaxReceiveBufferPerStream: 1}

		resp, err := client.Get("https://example.com/")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.ProtoMajor != 2 {
			t.Fatalf("protocol = %q, want HTTP/2", resp.Proto)
		}

		// The unread one-byte receive window leaves the handler blocked in Write.
		synctest.Wait()
		select {
		case err := <-writeDone:
			t.Fatalf("handler Write completed before timeout: %v", err)
		default:
		}

		synctest.Sleep(time.Minute - time.Nanosecond)
		select {
		case err := <-writeDone:
			t.Fatalf("handler Write completed before timeout: %v", err)
		default:
		}
		synctest.Sleep(time.Nanosecond)
		select {
		case err := <-writeDone:
			if err == nil {
				t.Fatal("handler Write returned no error after timeout")
			}
		default:
			t.Fatal("handler Write remained blocked after timeout")
		}
		if _, err := io.ReadAll(resp.Body); err == nil {
			t.Fatal("response body completed without a stream reset")
		}
	})
}

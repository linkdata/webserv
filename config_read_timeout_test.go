package webserv

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"testing/synctest"
	"time"
)

func TestDefaultHTTPServerReadTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		readErr := make(chan error, 1)
		handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			close(started)
			_, err := io.Copy(io.Discard, r.Body)
			readErr <- err
		})
		ts := httptest.NewTestServer(t, handler)
		ts.Config = defaultHTTPServer(handler)
		client := ts.Client()

		body, writer := io.Pipe()
		req, err := http.NewRequest(http.MethodPost, "http://example.com/", body)
		if err != nil {
			t.Fatal(err)
		}
		req.ContentLength = 1
		requestDone := make(chan struct{})
		go func() {
			defer close(requestDone)
			resp, _ := client.Do(req)
			if resp != nil {
				_ = resp.Body.Close()
			}
		}()

		<-started
		synctest.Sleep(time.Minute - time.Nanosecond)
		synctest.Wait()
		select {
		case err := <-readErr:
			t.Errorf("request body read ended early: %v", err)
		default:
		}
		synctest.Sleep(time.Nanosecond)
		synctest.Wait()
		select {
		case err := <-readErr:
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				t.Errorf("request body read error = %v, want deadline exceeded", err)
			}
		default:
			t.Error("request body read remained blocked after one minute")
		}

		_ = writer.Close()
		ts.CloseClientConnections()
		<-requestDone
	})
}

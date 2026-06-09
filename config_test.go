package webserv_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/linkdata/webserv"
)

const listeningLogMessage = "webserv: listening on"

var signalTestMu sync.Mutex

type notifyingLogger struct {
	logger *slog.Logger
	match  string
	ready  chan struct{}
	once   sync.Once
}

type panicListener struct{}

func (panicListener) Accept() (net.Conn, error) {
	panic(errors.New("accept panic"))
}

func (panicListener) Close() error {
	return nil
}

func (panicListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)}
}

func newNotifyingLogger(w io.Writer, match string) *notifyingLogger {
	return &notifyingLogger{
		logger: slog.New(slog.NewTextHandler(w, nil)),
		match:  match,
		ready:  make(chan struct{}),
	}
}

func (l *notifyingLogger) Info(msg string, keyValuePairs ...any) {
	l.logger.Info(msg, keyValuePairs...)
	if msg == l.match {
		l.once.Do(func() {
			close(l.ready)
		})
	}
}

func (l *notifyingLogger) Warn(msg string, keyValuePairs ...any) {
	l.logger.Warn(msg, keyValuePairs...)
}

func (l *notifyingLogger) Error(msg string, keyValuePairs ...any) {
	l.logger.Error(msg, keyValuePairs...)
}

func signalSelf(sig os.Signal) (err error) {
	var p *os.Process
	if p, err = os.FindProcess(os.Getpid()); err == nil {
		err = p.Signal(sig)
	}
	return
}

func signalWhenReady(ctx context.Context, ready <-chan struct{}, sig os.Signal) <-chan error {
	errc := make(chan error, 1)
	go func() {
		select {
		case <-ready:
			errc <- signalSelf(sig)
		case <-ctx.Done():
			errc <- ctx.Err()
		}
	}()
	return errc
}

func cancelWhenReady(ctx context.Context, ready <-chan struct{}, cancel context.CancelFunc) <-chan error {
	errc := make(chan error, 1)
	go func() {
		select {
		case <-ready:
			cancel()
			errc <- nil
		case <-ctx.Done():
			errc <- ctx.Err()
		}
	}()
	return errc
}

func TestConfig_ListenAndServe_Signalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process signalling from tests is not reliable on windows")
	}
	signalTestMu.Lock()
	defer signalTestMu.Unlock()

	withCertFiles(t, func(destdir string) {
		homeDir := os.Getenv("HOME")
		if st, err := os.Stat(homeDir); err != nil || !st.IsDir() {
			homeDir = ""
		}
		var buf bytes.Buffer
		logger := newNotifyingLogger(&buf, listeningLogMessage)
		cfg := &webserv.Config{
			Address:     "127.0.0.1:0",
			CertDir:     destdir,
			User:        os.Getenv("USER"),
			DataDir:     homeDir,
			DataDirMode: 0o750,
			Logger:      logger,
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Second*5)
		defer cancel()
		signalDone := signalWhenReady(ctx, logger.ready, syscall.SIGTERM)
		err := cfg.ListenAndServe(ctx, nil)
		if err != nil {
			t.Error(err)
			return
		}
		if err = <-signalDone; err != nil {
			t.Fatal(err)
		}
		s := buf.String()
		t.Log(s)
		if !strings.Contains(s, "signal") {
			t.Error("expected 'signal' in log output")
		}
		if !strings.Contains(s, "terminated") {
			t.Error("expected 'terminated' signal name in log output")
		}
	})
}

func TestConfig_ListenAndServe_Cancelled(t *testing.T) {
	withCertFiles(t, func(destdir string) {
		homeDir := os.Getenv("HOME")
		if st, err := os.Stat(homeDir); err != nil || !st.IsDir() {
			homeDir = ""
		}
		var buf bytes.Buffer
		logger := newNotifyingLogger(&buf, listeningLogMessage)
		cfg := &webserv.Config{
			Address:     "127.0.0.1:0",
			CertDir:     destdir,
			User:        os.Getenv("USER"),
			DataDir:     homeDir,
			DataDirMode: 0o750,
			Logger:      logger,
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Second*5)
		defer cancel()
		cancelDone := cancelWhenReady(ctx, logger.ready, cancel)
		err := cfg.ListenAndServe(ctx, nil)
		cancel()
		if cancelErr := <-cancelDone; cancelErr != nil && !errors.Is(cancelErr, context.Canceled) {
			t.Fatal(cancelErr)
		}
		if !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
		s := buf.String()
		t.Log(s)
		if !strings.Contains(s, "context canceled") {
			t.Error("expected 'context canceled' in log output")
		}
	})
}

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

	const noSuchUser = "webserv-no-such-user-listen"
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

func TestConfigServeWith_ExternalCloseReturnsPromptly(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	cfg := &webserv.Config{}
	accepted := make(chan struct{})
	var acceptedOnce sync.Once
	srv := &http.Server{
		ConnState: func(_ net.Conn, state http.ConnState) {
			if state == http.StateNew {
				acceptedOnce.Do(func() {
					close(accepted)
				})
			}
		},
	}
	ctx, cancel := context.WithTimeout(t.Context(), 750*time.Millisecond)
	defer cancel()

	closeDone := make(chan error, 1)
	go func() {
		var conn net.Conn
		var err error
		var d net.Dialer
		if conn, err = d.DialContext(ctx, "tcp", l.Addr().String()); err == nil {
			defer func() { _ = conn.Close() }()
			select {
			case <-accepted:
				err = srv.Close()
			case <-ctx.Done():
				err = ctx.Err()
			}
		}
		closeDone <- err
	}()

	start := time.Now()
	err = cfg.ServeWith(ctx, srv, l)
	elapsed := time.Since(start)
	if closeErr := <-closeDone; closeErr != nil {
		t.Fatal(closeErr)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ServeWith() blocked until context timeout (%v)", elapsed)
	}
	if elapsed > 350*time.Millisecond {
		t.Fatalf("ServeWith() took too long to return after external close: %v", elapsed)
	}
}

func TestConfigListen_ErrorClearsListenURL(t *testing.T) {
	const initialListenURL = "https://example.invalid:443"
	cfg := &webserv.Config{
		Address:   "127.0.0.1:99999",
		ListenURL: initialListenURL,
	}

	l, err := cfg.Listen()
	if l != nil {
		_ = l.Close()
	}
	if err == nil {
		t.Fatal("expected Listen() error")
	}
	if cfg.ListenURL != "" {
		t.Fatalf("ListenURL not cleared on error: got %q", cfg.ListenURL)
	}
}

func TestConfigListen_ErrorStillAbsolutizesCertDir(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	certDir := t.TempDir()
	relCertDir, err := filepath.Rel(cwd, certDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &webserv.Config{
		CertDir: relCertDir,
	}

	l, err := cfg.Listen()
	if l != nil {
		_ = l.Close()
	}
	if err == nil {
		t.Fatal("expected Listen() error for missing cert files")
	}
	absCertDir, err := filepath.Abs(relCertDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CertDir != absCertDir {
		t.Fatalf("CertDir not absolutized on error: got %q want %q", cfg.CertDir, absCertDir)
	}
}

func TestConfigListen_ErrorClearsCallerDataDir(t *testing.T) {
	// A failure before DataDir is computed (here, missing cert files in an
	// existing CertDir) still resets a caller-supplied cfg.DataDir to empty.
	cfg := &webserv.Config{
		CertDir: t.TempDir(),
		DataDir: t.TempDir(),
	}

	l, err := cfg.Listen()
	if l != nil {
		_ = l.Close()
	}
	if err == nil {
		t.Fatal("expected Listen() error for missing cert files")
	}
	if cfg.DataDir != "" {
		t.Fatalf("DataDir not cleared on error: got %q", cfg.DataDir)
	}
}

func assertPanics(t *testing.T, wantPanic string, fn func()) {
	t.Helper()
	defer func() {
		switch r := recover().(type) {
		case nil:
			t.Fatalf("expected call to panic with %q", wantPanic)
		case string:
			if r != wantPanic {
				t.Fatalf("panic = %q, want %q", r, wantPanic)
			}
		default:
			t.Fatalf("panic = %v (%T), want string %q", r, r, wantPanic)
		}
	}()
	fn()
}

func TestConfigListenAndServe_NilContextPanics(t *testing.T) {
	cfg := &webserv.Config{}
	var nilCtx context.Context
	assertPanics(t, "webserv: nil context.Context", func() {
		_ = cfg.ListenAndServe(nilCtx, nil)
	})
}

func TestConfigServeWith_NilContextPanics(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	cfg := &webserv.Config{}
	var nilCtx context.Context
	assertPanics(t, "webserv: nil context.Context", func() {
		_ = cfg.ServeWith(nilCtx, &http.Server{}, l)
	})
}

func TestConfigServeWith_NilListenerPanics(t *testing.T) {
	cfg := &webserv.Config{}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assertPanics(t, "webserv: nil net.Listener", func() {
		_ = cfg.ServeWith(ctx, &http.Server{}, nil)
	})
}

func TestConfigServeWith_NilServerPanics(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	cfg := &webserv.Config{}
	assertPanics(t, "webserv: nil http.Server", func() {
		_ = cfg.ServeWith(t.Context(), nil, l)
	})
}

func TestConfigServeWith_RecoversServePanic(t *testing.T) {
	cfg := &webserv.Config{}
	srv := &http.Server{}

	err := cfg.ServeWith(t.Context(), srv, panicListener{})
	if err == nil {
		t.Fatal("expected ServeWith() error")
	}
	if !errors.Is(err, webserv.ErrServePanic) {
		t.Fatalf("ServeWith() error = %v, want match %v", err, webserv.ErrServePanic)
	}
	if errors.Unwrap(err) == nil {
		t.Fatalf("ServeWith() error = %v, expected non-nil unwrap for recovered panic error", err)
	}
}

func TestConfigServeWith_FiltersTLSHandshakeErrors(t *testing.T) {
	withCertFiles(t, func(destdir string) {
		logs := servePlainHTTPToTLS(t, destdir, false)
		if strings.Contains(logs, "TLS handshake error") {
			t.Fatalf("ServeWith() logged filtered TLS handshake error: %q", logs)
		}
	})
}

func TestConfigServeWith_LogTLSErrorsForwardsTLSHandshakeErrors(t *testing.T) {
	withCertFiles(t, func(destdir string) {
		logs := servePlainHTTPToTLS(t, destdir, true)
		if !strings.Contains(logs, "TLS handshake error") {
			t.Fatalf("ServeWith() did not log TLS handshake error: %q", logs)
		}
	})
}

func servePlainHTTPToTLS(t *testing.T, certDir string, logTLSErrors bool) (logs string) {
	t.Helper()

	cfg := &webserv.Config{
		Address:      "127.0.0.1:0",
		CertDir:      certDir,
		LogTLSErrors: logTLSErrors,
	}
	l, err := cfg.Listen()
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	previousErrorLog := log.New(&buf, "", 0)
	srv := &http.Server{
		ErrorLog: previousErrorLog,
	}
	defer func() { _ = srv.Close() }()

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- cfg.ServeWith(ctx, srv, l)
	}()

	conn, err := net.DialTimeout("tcp", l.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err = conn.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err = io.WriteString(conn, "GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(conn)
	if closeErr := conn.Close(); readErr == nil {
		readErr = closeErr
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(body), "Client sent an HTTP request to an HTTPS server") {
		t.Fatalf("plain HTTP response = %q", string(body))
	}

	if err = srv.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ServeWith()")
	}
	if err != nil {
		t.Fatal(err)
	}
	// With LogTLSErrors set, ServeWith leaves srv.ErrorLog untouched. Otherwise
	// it installs the filtering logger once (wrapping the previous one) and
	// keeps it in place for the lifetime of the call.
	if logTLSErrors {
		if srv.ErrorLog != previousErrorLog {
			t.Fatal("ServeWith() replaced ErrorLog despite LogTLSErrors")
		}
	} else if srv.ErrorLog == previousErrorLog {
		t.Fatal("ServeWith() did not install the TLS handshake error filter")
	}
	logs = buf.String()
	return
}

func TestConfigListen_ErrorAfterBindMayPopulateListenURL(t *testing.T) {
	const noSuchUser = "webserv-no-such-user-listenurl"
	if _, err := user.Lookup(noSuchUser); err == nil {
		t.Skipf("test user unexpectedly exists: %q", noSuchUser)
	}

	cfg := &webserv.Config{
		Address: "127.0.0.1:0",
		User:    noSuchUser,
	}

	l, err := cfg.Listen()
	if l != nil {
		_ = l.Close()
	}
	if err == nil {
		t.Fatal("expected Listen() error")
	}
	if cfg.ListenURL == "" {
		t.Fatal("expected ListenURL to be populated after bind, even though Listen() failed later")
	}
}

func TestConfigListen_EmptyDataDirStaysEmpty(t *testing.T) {
	cfg := &webserv.Config{
		Address: "127.0.0.1:0",
	}

	l, err := cfg.Listen()
	if l != nil {
		_ = l.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != "" {
		t.Fatalf("expected empty DataDir, got %q", cfg.DataDir)
	}
}

func TestConfigServeWith_PropagatesShutdownContextError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process signalling from tests is not reliable on windows")
	}
	signalTestMu.Lock()
	defer signalTestMu.Unlock()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	started := make(chan struct{})
	unblock := make(chan struct{})
	reqDone := make(chan struct{})

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			close(started)
			<-unblock
			_, _ = w.Write([]byte("ok"))
		}),
	}

	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()

	go func() {
		defer close(reqDone)
		_, _ = http.Get("http://" + l.Addr().String())
	}()

	signalDone := signalWhenReady(ctx, started, syscall.SIGTERM)
	err = (&webserv.Config{}).ServeWith(ctx, srv, l)
	if signalErr := <-signalDone; signalErr != nil {
		t.Fatal(signalErr)
	}
	close(unblock)
	<-reqDone

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ServeWith() error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestConfigServeWith_SignalShutdownUsesConfigShutdownTimeLimit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process signalling from tests is not reliable on windows")
	}
	signalTestMu.Lock()
	defer signalTestMu.Unlock()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	started := make(chan struct{})
	unblock := make(chan struct{})
	reqDone := make(chan struct{})
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			close(started)
			<-unblock
			_, _ = w.Write([]byte("ok"))
		}),
	}
	defer func() { _ = srv.Close() }()

	const shutdownTimeLimit = time.Millisecond * 10
	cfg := &webserv.Config{
		ShutdownTimeLimit: shutdownTimeLimit,
	}

	done := make(chan error, 1)
	go func() {
		done <- cfg.ServeWith(t.Context(), srv, l)
	}()

	go func() {
		defer close(reqDone)
		client := &http.Client{Timeout: 2 * shutdownTimeLimit}
		_, _ = client.Get("http://" + l.Addr().String())
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for handler to start")
	}

	if err = signalSelf(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	select {
	case err = <-done:
	case <-time.After(1500 * time.Millisecond):
		close(unblock)
		<-reqDone
		t.Fatal("ServeWith() did not return within 1.5s after SIGTERM")
	}

	close(unblock)
	<-reqDone

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ServeWith() error = %v, want %v", err, context.DeadlineExceeded)
	}
}

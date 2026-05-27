package webserv

import (
	"context"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const defaultShutdownTimeLimit = time.Second

// Config contains the startup and serving settings for a simple web service.
//
// The zero value is usable: Listen serves HTTP on the default address and port,
// Serve uses a default http.Server, no user switch or data directory setup is
// performed, and no logs are emitted.
type Config struct {
	Address              string        // optional specific address to listen on; use ":port" for port-only
	CertDir              string        // if set, directory to look for fullchain.pem and privkey.pem
	FullchainPem         string        // set to override filename for "fullchain.pem"
	PrivkeyPem           string        // set to override filename for "privkey.pem"
	User                 string        // if set, user to switch to after opening listening port
	DataDir              string        // if set, the data directory to use (created only when DataDirMode is nonzero); if unset, may be filled in after Listen
	DefaultDataDirSuffix string        // if set and DataDir is not set, set DataDir to the user's default data dir plus this suffix
	DataDirMode          fs.FileMode   // if nonzero, create DataDir if it does not exist using this mode
	ListenURL            string        // if set, the external URL clients can reach us at. If unset, Listen may fill this in (e.g. "https://localhost:8443"), even when Listen later returns an error after binding.
	ShutdownTimeLimit    time.Duration // maximum time ServeWith waits for graceful shutdown; zero uses a 1 second default
	LogTLSErrors         bool          // if set, http.Server TLS handshake error messages are not filtered
	Logger               Logger        // logger to use, if nil logs nothing
}

func (cfg *Config) logInfo(msg string, keyValuePairs ...any) {
	if cfg.Logger != nil {
		cfg.Logger.Info("webserv: "+msg, keyValuePairs...)
	}
}

func (cfg *Config) shutdownTimeLimit() (limit time.Duration) {
	if limit = cfg.ShutdownTimeLimit; limit == 0 {
		limit = defaultShutdownTimeLimit
	}
	return
}

// Listen performs initial setup for a simple web server and returns a
// net.Listener if successful.
//
// First it loads certificates if cfg.CertDir is set, and then starts a net.Listener
// (TLS or normal). The listener will default to all addresses and standard port
// depending on privileges and if a certificate was loaded or not.
//
// If cfg.Address was set, any address or port given there overrides these defaults.
//
// If cfg.User is set it then switches to that user and the users primary group.
// Note that this is not supported on Windows.
//
// If cfg.DataDir or cfg.DefaultDataDirSuffix is set, calculates the absolute
// data directory path and sets cfg.DataDir. If cfg.DataDirMode is nonzero, the
// directory will be created if necessary.
//
// On return, cfg.CertDir and cfg.DataDir will be absolute paths or be empty.
// If cfg.ListenURL was empty it may be set to a best-guess printable and connectable
// URL like "http://localhost:80" as soon as the socket is opened.
// Therefore cfg.ListenURL can be non-empty even if Listen returns an error from a
// later step, such as user switching or data directory setup.
func (cfg *Config) Listen() (l net.Listener, err error) {
	if l, cfg.ListenURL, cfg.CertDir, err = Listener(cfg.Address, cfg.CertDir, cfg.FullchainPem, cfg.PrivkeyPem, cfg.ListenURL); err == nil {
		if cfg.CertDir != "" {
			cfg.logInfo("loaded certificates", "dir", cfg.CertDir)
		}
		if err = BecomeUser(cfg.User); err == nil {
			if cfg.User != "" {
				cfg.logInfo("user switched", "user", cfg.User)
			}
			if cfg.DataDir, err = DefaultDataDir(cfg.DataDir, cfg.DefaultDataDirSuffix); err == nil {
				if cfg.DataDir, err = UseDataDir(cfg.DataDir, cfg.DataDirMode); err == nil {
					if cfg.DataDir != "" {
						cfg.logInfo("data directory", "dir", cfg.DataDir)
					}
				}
			}
		}
	}
	if err != nil {
		cfg.DataDir = ""
		if l != nil {
			_ = l.Close()
			l = nil
		}
	}
	return
}

// ServeWith sets up a signal handler to catch SIGINT and SIGTERM and then calls srv.Serve(l).
// If ctx is canceled, the server will be shut down and this function returns with ctx.Err().
//
// Returns nil if the server started successfully and then cleanly shut down.
// Graceful shutdown waits for cfg.ShutdownTimeLimit, or 1 second when
// cfg.ShutdownTimeLimit is zero.
//
// Panics if ctx, srv or l is nil. Panics from srv.Serve are recovered and
// returned as an error matching ErrServePanic.
func (cfg *Config) ServeWith(ctx context.Context, srv *http.Server, l net.Listener) (err error) {
	if ctx == nil {
		panic("webserv: nil context.Context")
	}
	if srv == nil {
		panic("webserv: nil http.Server")
	}
	if l == nil {
		panic("webserv: nil net.Listener")
	}
	serveErr := make(chan error, 1)
	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg.logInfo("listening on", "address", l.Addr(), "url", cfg.ListenURL)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				serveErr <- newErrServePanic(p)
			}
		}()
		serveErr <- func() (err error) {
			if !cfg.LogTLSErrors {
				restore := filterTLSErrorLog(srv)
				defer restore()
			}
			err = srv.Serve(l)
			return
		}()
	}()
	select {
	case err = <-serveErr:
	case <-sigCtx.Done():
		err = ctx.Err()
		var reason error
		if reason = context.Cause(ctx); reason == nil {
			reason = context.Cause(sigCtx)
		}
		cfg.logInfo("stopped", "reason", reason)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.shutdownTimeLimit())
		shutdownErr := srv.Shutdown(shutdownCtx)
		shutdownCancel()
		serveExitErr := <-serveErr
		if err == nil {
			if shutdownErr != nil {
				err = shutdownErr
			} else {
				err = serveExitErr
			}
		}
	}
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	return err
}

// Serve creates an http.Server with reasonable defaults and calls ServeWith.
//
// The server uses handler as its Handler; if handler is nil, http.DefaultServeMux
// is used by net/http. ReadHeaderTimeout is set to 5 seconds and IdleTimeout is
// set to 1 minute.
//
// Serve takes ownership of l for serving. It returns nil after a clean shutdown,
// returns ctx.Err() when ctx cancellation starts shutdown, and otherwise returns
// the error from serving or shutting down.
//
// Panics if ctx or l is nil.
func (cfg *Config) Serve(ctx context.Context, l net.Listener, handler http.Handler) error {
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: time.Second * 5,
		IdleTimeout:       time.Minute,
	}
	return cfg.ServeWith(ctx, srv, l)
}

// ListenAndServe calls Listen followed by Serve.
//
// It returns ctx.Err() without opening a listener if ctx is already canceled.
// Otherwise, it performs the setup documented by Listen and then serves requests
// with the default server settings documented by Serve.
//
// The returned error is from Listen, Serve, ctx cancellation, or shutdown. A nil
// return means the server started successfully and then shut down cleanly.
//
// Panics if ctx is nil.
func (cfg *Config) ListenAndServe(ctx context.Context, handler http.Handler) (err error) {
	if err = ctx.Err(); err == nil {
		var l net.Listener
		if l, err = cfg.Listen(); err == nil {
			err = cfg.Serve(ctx, l, handler)
		}
	}
	return
}

package webserv

import (
	"context"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var ShutdownTimeLimit = time.Second

type Config struct {
	Address              string      // optional specific address (and/or port) to listen on
	CertDir              string      // if set, directory to look for fullchain.pem and privkey.pem
	FullchainPem         string      // set to override filename for "fullchain.pem"
	PrivkeyPem           string      // set to override filename for "privkey.pem"
	User                 string      // if set, user to switch to after opening listening port
	DataDir              string      // if set, create this directory, if unset will be filled in after Listen
	DefaultDataDirSuffix string      // if set and DataDir is not set, set DataDir to the user's default data dir plus this suffix
	DataDirMode          fs.FileMode // if nonzero, create DataDir if it does not exist using this mode
	ListenURL            string      // if set, the external URL clients can reach us at. If unset, Listen may fill this in (e.g. "https://localhost:8443"), even when Listen later returns an error after binding.
	Logger               Logger      // logger to use, if nil logs nothing
}

func (cfg *Config) logInfo(msg string, keyValuePairs ...any) {
	if cfg.Logger != nil && len(keyValuePairs) > 1 {
		s, ok := keyValuePairs[1].(string)
		if !(ok && s == "") {
			cfg.Logger.Info("webserv: "+msg, keyValuePairs...)
		}
	}
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
		cfg.logInfo("loaded certificates", "dir", cfg.CertDir)
		if err = BecomeUser(cfg.User); err == nil {
			cfg.logInfo("user switched", "user", cfg.User)
			if cfg.DataDir, err = DefaultDataDir(cfg.DataDir, cfg.DefaultDataDirSuffix); err == nil {
				if cfg.DataDir, err = UseDataDir(cfg.DataDir, cfg.DataDirMode); err == nil {
					cfg.logInfo("data directory", "dir", cfg.DataDir)
				}
			}
		}
		if err != nil {
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
//
// Panics if any of the arguments are nil.
func (cfg *Config) ServeWith(ctx context.Context, srv *http.Server, l net.Listener) (err error) {
	serveErr := make(chan error, 1)
	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg.logInfo("listening on", "address", l.Addr(), "url", cfg.ListenURL)
	go func() {
		serveErr <- srv.Serve(l)
	}()
	select {
	case err = <-serveErr:
	case <-sigCtx.Done():
		reason := "interrupted"
		if err = ctx.Err(); err != nil {
			reason = err.Error()
		} else {
			if cause := context.Cause(sigCtx); cause != nil {
				reason = cause.Error()
			}
		}
		cfg.logInfo("stopped", "reason", reason)
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, ShutdownTimeLimit)
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
	if err == http.ErrServerClosed {
		err = nil
	}
	return err
}

// Serve creates a http.Server with reasonable defaults and calls ServeWith.
func (cfg *Config) Serve(ctx context.Context, l net.Listener, handler http.Handler) error {
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: time.Second * 5,
	}
	return cfg.ServeWith(ctx, srv, l)
}

// ListenAndServe calls Listen followed by Serve.
func (cfg *Config) ListenAndServe(ctx context.Context, handler http.Handler) (err error) {
	if err = ctx.Err(); err == nil {
		var l net.Listener
		if l, err = cfg.Listen(); err == nil {
			err = cfg.Serve(ctx, l, handler)
		}
	}
	return
}

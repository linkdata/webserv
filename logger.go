package webserv

// Logger matches log/slog.Info(), Warn() and Error(), but allows one to use another logger using an adaptor.
type Logger interface {
	Info(msg string, keyValuePairs ...any)
	Warn(msg string, keyValuePairs ...any)
	Error(msg string, keyValuePairs ...any)
}

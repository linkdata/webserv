package webserv

// InfoLogger matches log/slog.Info(), but allows one to use another logger using an adaptor.
type InfoLogger interface {
	Info(msg string, keyValuePairs ...any)
}

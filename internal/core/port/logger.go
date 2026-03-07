package port

// Logger abstracts structured logging.
type Logger interface {
	Printf(format string, args ...any)
}

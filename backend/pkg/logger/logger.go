package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog.Logger with convenience methods.
type Logger struct {
	zerolog.Logger
}

// Options configures the logger.
type Options struct {
	Level     string // debug, info, warn, error, fatal
	Format    string // json, console
	Output    io.Writer
	Component string // optional component name
}

// New creates a new Logger with the given options.
func New(level string) zerolog.Logger {
	return NewWithOptions(Options{Level: level, Format: "console"}).Logger
}

// NewWithOptions creates a new Logger with full configuration.
func NewWithOptions(opts Options) *Logger {
	lvl, err := zerolog.ParseLevel(opts.Level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	var output io.Writer
	if opts.Output != nil {
		output = opts.Output
	} else {
		output = os.Stdout
	}

	if opts.Format != "json" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
		}
	}

	ctx := zerolog.New(output).
		Level(lvl).
		With().
		Timestamp().
		Caller()

	if opts.Component != "" {
		ctx = ctx.Str("component", opts.Component)
	}

	return &Logger{Logger: ctx.Logger()}
}

// WithComponent returns a new Logger with the component field set.
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With().Str("component", component).Logger(),
	}
}

// WithRequestID returns a new Logger with the request ID field set.
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger: l.Logger.With().Str("request_id", requestID).Logger(),
	}
}

// WithField returns a new Logger with an additional field.
func (l *Logger) WithField(key string, value any) *Logger {
	return &Logger{
		Logger: l.Logger.With().Interface(key, value).Logger(),
	}
}

// WithFields returns a new Logger with multiple additional fields.
func (l *Logger) WithFields(fields map[string]any) *Logger {
	ctx := l.Logger.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &Logger{Logger: ctx.Logger()}
}

// Global is the default global logger instance.
var Global = NewWithOptions(Options{Level: "info", Format: "console"})

// SetGlobal replaces the global logger.
func SetGlobal(l *Logger) {
	Global = l
}

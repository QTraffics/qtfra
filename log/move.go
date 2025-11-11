package log

import (
	"context"
	"log/slog"
	"strings"
)

type Handler = slog.Handler

// Logger provides a basic implementation for logging, sufficient for most use cases to record information.
// `Every` custom Logger implementation should adhere to this interface.
// Additionally, any extensions built upon the basic Logger, such as ContextLogger, should also implement this interface.
type Logger interface {
	// Enabled checks if logging is active for the given level in the provided context.
	Enabled(ctx context.Context, level Level) bool
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// ContextLogger extends the Logger interface by adding methods that accept a context parameter.
// This allows for passing additional contextual information to the handler during logging operations,
// which can be useful for carrying extra details like request IDs or user sessions.
//
// Note that slog.Logger implementations should satisfy both Logger and ContextLogger interfaces.
type ContextLogger interface {
	Logger // Embeds the base Logger interface.
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

type UnimplementedContextLogger struct{ Logger }

func (n *UnimplementedContextLogger) DebugContext(ctx context.Context, msg string, args ...any) {
	n.Logger.Debug(msg, args...)
}

func (n *UnimplementedContextLogger) InfoContext(ctx context.Context, msg string, args ...any) {
	n.Logger.Info(msg, args...)
}

func (n *UnimplementedContextLogger) WarnContext(ctx context.Context, msg string, args ...any) {
	n.Logger.Warn(msg, args...)
}

func (n *UnimplementedContextLogger) ErrorContext(ctx context.Context, msg string, args ...any) {
	n.Logger.Error(msg, args...)
}

func AsContextLogger(l Logger) ContextLogger {
	if cl, ok := l.(ContextLogger); ok {
		return cl
	}

	return &UnimplementedContextLogger{Logger: l}
}

var (
	_ Logger        = (*slog.Logger)(nil)
	_ ContextLogger = (*slog.Logger)(nil)
)

type handler interface {
	Handler() Handler
}

type with interface {
	With(...any) Logger
}

type group interface {
	WithGroup(string) Logger
}

func New(handler Handler, opt ...any) ContextLogger {
	return newFeatureLogger(handler, opt...)
}

func NewSlog(handler Handler) *slog.Logger {
	return slog.New(handler)
}

func GetHandler(l Logger) Handler {
	if hh, ok := l.(handler); ok {
		return hh.Handler()
	}
	return nil
}

func With(l Logger, v ...any) Logger {
	if len(v) == 0 {
		return l
	}

	if ww, ok := l.(with); ok {
		return ww.With(v...)
	}

	if ww, ok := l.(*slog.Logger); ok {
		noOption := make([]any, 0, len(v))
		for _, vv := range v {
			switch x := vv.(type) {
			case Option:
			default:
				noOption = append(noOption, x)
			}
		}
		if len(noOption) == 0 {
			return ww
		}
		return ww.With(noOption...)
	}
	panic("Logger doesn't implement With")
}

func WithGroup(l Logger, name string) Logger {
	if len(name) == 0 {
		return l
	}

	if ww, ok := l.(group); ok {
		return ww.WithGroup(name)
	}

	if ww, ok := l.(*slog.Logger); ok {
		return ww.WithGroup(name)
	}
	panic("Logger doesn't implement group")
}

type Level = slog.Level

const (
	LevelDebug   = slog.LevelDebug
	LevelInfo    = slog.LevelInfo
	LevelWarn    = slog.LevelWarn
	LevelError   = slog.LevelError
	LevelDisable = slog.LevelError + 1
)

func ParseLevel(s string) (l Level, ok bool) {
	s = strings.TrimSpace(s)
	us := strings.ToUpper(s)
	switch us {
	case "DEBUG":
		return LevelDebug, true
	case "INFO":
		return LevelInfo, true
	case "WARN", "WARNING":
		return LevelWarn, true
	case "ERROR":
		return LevelError, true
	case "OFF", "QUIET", "DISABLED":
		return LevelDisable, true
	default:
		return defaultVal[Level](), false
	}
}

func defaultVal[T any]() T {
	var vv T
	return vv
}

func attrsToAny(attrs ...slog.Attr) []any {
	ans := make([]any, 0, len(attrs))
	for _, v := range attrs {
		ans = append(ans, v)
	}
	return ans
}

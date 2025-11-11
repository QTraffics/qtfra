package log

import (
	"log/slog"
)

type Factory struct {
	handler slog.Handler
}

// Deprecated: useless
func NewFactory(handler Handler) *Factory {
	return &Factory{handler: handler}
}

func (f *Factory) internalNew() *slog.Logger {
	return slog.New(f.handler)
}

func (f *Factory) New(vv ...any) ContextLogger {
	slogLogger := f.internalNew()
	var attrs []any
	for _, v := range vv {
		switch x := v.(type) {
		case string:
			if len(attrs) != 0 {
				slogLogger = slogLogger.With(attrs...)
				attrs = nil
			}
			slogLogger = slogLogger.WithGroup(x)
		case slog.Attr:
			attrs = append(attrs, x)
		default:
			panic("unexcepted input")
		}
	}
	slogLogger = slogLogger.With(attrs...)

	return slogLogger
}

package log

import (
	"log/slog"
)

const (
	KeyError = "error"
)

func AttrError(err error) slog.Attr {
	if err == nil {
		panic("log error on a nil error")
	}

	return slog.Any(KeyError, ValueFunc(func() slog.Value {
		return slog.StringValue(err.Error())
	}))
}

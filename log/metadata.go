package log

import (
	"log/slog"
	"strconv"
	"strings"
)

const (
	metadataPrefix   = "_builtin.enhancements.metadata"
	metadataSplitter = ":"
)

var _ slog.LogValuer = (ValueFunc)(nil)

type ValueFunc func() slog.Value

func (v ValueFunc) LogValue() slog.Value {
	return v()
}

func NewFixedMetadata(v any) slog.Attr {
	key := buildAnonymousKey()
	return NewMetadata(key, v)
}

func NewMetadata(name string, v any) slog.Attr {
	key := strings.Join([]string{metadataPrefix, name}, metadataSplitter)
	var value slog.Value
	switch x := v.(type) {
	case string:
		value = slog.StringValue(x)
	case slog.Attr:
		value = x.Value
	case slog.Value:
		value = x
	default:
		value = slog.AnyValue(v)
	}

	return slog.Attr{Key: key, Value: value}
}

func SplitMetadata(attrs []slog.Attr) (meta, extra []slog.Attr) {
	for _, attr := range attrs {
		splitN := strings.SplitN(attr.Key, metadataSplitter, 2)
		if len(splitN) < 2 || splitN[0] != metadataPrefix {
			extra = append(extra, attr)
			continue
		}
		attr.Key = splitN[1]
		meta = append(meta, attr)
	}
	return
}

var anonymousKey uint64

func buildAnonymousKey() string {
	anonymousKey++
	return "anonymous" + "." + strconv.FormatUint(anonymousKey, 10)
}

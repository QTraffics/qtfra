package loghandler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/QTraffics/qtfra/buf"
	"github.com/QTraffics/qtfra/enhancements/iolib"
	"github.com/QTraffics/qtfra/enhancements/maplib"
	"github.com/QTraffics/qtfra/enhancements/slicelib"
	"github.com/QTraffics/qtfra/ex"
	"github.com/QTraffics/qtfra/log"
	"github.com/QTraffics/qtfra/values"
)

var (
	internalHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})
	internal log.Logger = slog.New(internalHandler)
)

var (
	DefaultConsoleHandler log.Handler = NewConsoleHandler(os.Stderr,
		ConsoleHandlerOption{EnableTime: true, SourceLevel: log.LevelError, Level: log.LevelDebug, LevelFormatter: log.ColorLevelFormatter}).
		WithAttrs([]slog.Attr{log.NewFixedMetadata("Default")})
)

var (
	space             byte = ' '
	dot               byte = '.'
	defaultBufferSize      = 16384
)

var _ log.Handler = (*ConsoleHandler)(nil)

type ConsoleHandler struct {
	level, sourceLevel log.Level
	enableTime         bool
	timeFormat         func(t time.Time) string
	levelFormat        func(l log.Level) string

	// the writer should be thread safe.
	// (implement threads.Safe)
	writer io.Writer

	// internal elements
	groupPrefix     []byte
	preFormatedAttr []byte
	metadata        []slog.Attr
}

type ConsoleHandlerOption struct {
	Level       log.Level
	SourceLevel log.Level
	EnableTime  bool

	TimeFormatter  func(t time.Time) string
	LevelFormatter func(level log.Level) string
}

func NewConsoleHandler(w io.Writer, option ConsoleHandlerOption) log.Handler {
	if w == nil || w == io.Discard {
		return slog.DiscardHandler
	}
	option.TimeFormatter = values.UseDefaultNil(option.TimeFormatter, log.RFC3339TimeFormatter)
	option.LevelFormatter = values.UseDefaultNil(option.LevelFormatter, log.EqualLengthLevelFormatter)

	h := &ConsoleHandler{
		writer:      iolib.NewSafeWriter(w),
		level:       option.Level,
		sourceLevel: option.SourceLevel,
		enableTime:  option.EnableTime,
		timeFormat:  option.TimeFormatter,
		levelFormat: option.LevelFormatter,
	}
	return h
}

func (h *ConsoleHandler) newState() *consoleHandlerState {
	return &consoleHandlerState{
		buffer: iolib.NewBufWriter(h.writer, buf.NewSize(defaultBufferSize)),
		group:  h.groupPrefix,
		level:  h.levelFormat,
		time:   h.timeFormat,
	}
}

func (h *ConsoleHandler) clone() *ConsoleHandler {
	return &ConsoleHandler{
		level:       h.level,
		sourceLevel: h.sourceLevel,
		enableTime:  h.enableTime,
		timeFormat:  h.timeFormat,
		levelFormat: h.levelFormat,

		writer:          h.writer,
		groupPrefix:     slices.Clone(h.groupPrefix),
		preFormatedAttr: slices.Clone(h.preFormatedAttr),
		metadata:        slices.Clone(h.metadata),
	}
}

func (h *ConsoleHandler) Enabled(ctx context.Context, level log.Level) bool {
	return h.level <= level
}

func (h *ConsoleHandler) Handle(ctx context.Context, r slog.Record) error {
	state := h.newState()
	defer state.Free()

	// time
	if h.enableTime {
		if r.Time.IsZero() {
			r.Time = time.Now()
		}
		if state.WriteTime(r.Time); state.Err != nil {
			return ex.Cause(state.Err, "write time")
		}
	}

	// level
	if h.enableTime {
		state.Space()
	}
	if state.WriteLevel(r.Level); state.Err != nil {
		return ex.Cause(state.Err, "write level")
	}

	// metadata
	for i, m := range h.metadata {
		if state.WriteMeta(m); state.Err != nil {
			return ex.Cause(state.Err, fmt.Sprintf("write metadata for: index: %d , key: %s", i, m))
		}
	}

	// message
	state.Space()
	if state.WriteString(r.Message); state.Err != nil {
		return ex.Cause(state.Err, "write message")
	}
	// attrs

	if len(h.preFormatedAttr) != 0 {
		state.Space()
		if state.Write(h.preFormatedAttr); state.Err != nil {
			return ex.Cause(state.Err, "write attrs")
		}
	}

	r.Attrs(state.WriteAttr)
	if state.Err != nil {
		return ex.Cause(state.Err, "write attrs")
	}

	// next line
	if state.NextLine(); state.Err != nil {
		return ex.Cause(state.Err, "next line")
	}

	// sources
	if r.Level >= h.sourceLevel {
		if source := r.Source(); source != nil {
			state.Source(source)
		}
	}

	return state.Err
}

func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	metadata, extraAttr := log.SplitMetadata(attrs)
	h2 := h.clone()
	if len(metadata) != 0 {
		h2.withMetadata(metadata)
	}
	if len(extraAttr) != 0 {
		h2.withAttrs(extraAttr)
	}

	return h2
}

func (h *ConsoleHandler) withMetadata(attrs []slog.Attr) {
	attrs = slicelib.UniqByLast(attrs, func(it slog.Attr) string {
		return it.Key
	})

	if len(h.metadata) == 0 {
		h.metadata = attrs
		return
	}

	indexes := maplib.IndexMap(slicelib.Map(h.metadata, func(it slog.Attr) string {
		return it.Key
	}))

	for i := 0; i < len(attrs); i++ {
		attr := attrs[i]
		if old, ok := indexes[attr.Key]; ok && old < len(h.metadata) {
			h.metadata[old] = attr
		} else {
			h.metadata = append(h.metadata, attr)
		}
	}
}

func (h *ConsoleHandler) withAttrs(attrs []slog.Attr) {
	var (
		group     string
		attrBytes [][]byte
	)
	if h.groupPrefix != nil {
		group = string(h.groupPrefix)
	}
	if len(h.preFormatedAttr) != 0 {
		attrBytes = append(attrBytes, h.preFormatedAttr)
	}

	for i := 0; i < len(attrs); i++ {
		attr := attrs[i]
		key := attr.Key
		if len(group) != 0 {
			key = strings.Join([]string{group, key}, string(dot))
		}
		attr.Key = key
		attrBytes = append(attrBytes, []byte(attr.String()))
	}
	if len(attrBytes) != 0 {
		h.preFormatedAttr = bytes.Join(attrBytes, []byte{' '})
	}

}

func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.withGroup(name)
	return h2
}

func (h *ConsoleHandler) withGroup(name string) {
	if len(h.groupPrefix) == 0 {
		h.groupPrefix = []byte(name)
		return
	}
	h.groupPrefix = bytes.Join([][]byte{h.groupPrefix, []byte(name)}, []byte{dot})
}

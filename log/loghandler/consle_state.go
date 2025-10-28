package loghandler

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/QTraffics/qtfra/enhancements/iolib"
	"github.com/QTraffics/qtfra/log"
	"github.com/QTraffics/qtfra/sys/sysvars"
)

type consoleHandlerState struct {
	Err error

	buffer *iolib.BufWriter
	group  []byte

	level func(l log.Level) string
	time  func(t time.Time) string
}

func (s *consoleHandlerState) Free() {
	defer func() {
		if r := recover(); r != nil {
			internal.Error("panic during flush",
				slog.Any("panic", r))
		}
	}()

	if s.buffer != nil {
		if err := s.buffer.Flush(); err != nil {
			internal.Error("flush log buffer failed",
				log.AttrError(err))
		}
		s.buffer.Free()
	}
}

func (s *consoleHandlerState) Write(p []byte) {
	if s.Err != nil {
		return
	}
	_, s.Err = s.buffer.Write(p)
}

func (s *consoleHandlerState) WriteString(ss string) {
	if s.Err != nil {
		return
	}
	_, s.Err = s.buffer.WriteString(ss)
}

func (s *consoleHandlerState) WriteTime(t time.Time) {
	if s.Err != nil {
		return
	}
	_, s.Err = s.buffer.WriteString(s.time(t))
}

func (s *consoleHandlerState) WriteLevel(l log.Level) {
	if s.Err != nil {
		return
	}
	_, s.Err = s.buffer.WriteString(s.level(l))
}

func (s *consoleHandlerState) WriteAttr(attr slog.Attr) bool {
	if s.Err != nil {
		return false
	}

	if s.Space(); s.Err != nil {
		return false
	}

	if attr.Key == "" {
		attr.Key = "!BADKEY"
	}

	value := attr.Value.Resolve()
	if value.Kind() == slog.KindGroup {
		for _, v := range value.Group() {
			v.Key = attr.Key + "." + v.Key // apply the old key
			if !s.WriteAttr(v) {
				return false
			}
		}
		return true
	}

	if len(s.group) != 0 {
		s.writeElement(string(s.group), &dot)
		if s.Err != nil {
			return false
		}
	}
	var valueStr string
	if value.Kind() == slog.KindTime && s.time != nil {
		valueStr = s.time(value.Time())
	} else {
		valueStr = value.String()
	}

	if strings.Contains(valueStr, " ") {
		valueStr = "`" + valueStr + "`"
	}
	s.writeElement(attr.Key+"="+valueStr, &space)
	return s.Err == nil
}

func (s *consoleHandlerState) Source(source *slog.Source) {
	if s.Err != nil {
		return
	}
	var out string
	if sysvars.DebugEnabled {
		out = fmt.Sprintf("Caller: %s:%d %s \n", source.File, source.Line, source.Function)
	} else {
		out = fmt.Sprintf("Caller: %s \n", source.Function)
	}

	s.writeElement(out, nil)
}

func (s *consoleHandlerState) NextLine() {
	if s.Err != nil {
		return
	}
	s.Err = s.buffer.WriteByte('\n')
}

func (s *consoleHandlerState) Space() {
	if s.Err != nil {
		return
	}
	s.Err = s.buffer.WriteByte(space)
}

func (s *consoleHandlerState) WriteMeta(meta slog.Attr) {
	if s.Err != nil {
		return
	}

	if s.Space(); s.Err != nil {
		return
	}

	value := meta.Value.Resolve()

	if value.Kind() == slog.KindGroup {
		values := groupValues(value)
		s.writeElement("["+strings.Join(values, " ")+"]", nil)
		return
	}

	str := value.String()
	if len(str) == 0 {
		str = "!EMPTY"
	}
	s.writeElement("["+str+"]", nil)
}

// writeElement writes string and optional suffix byte to buffer
func (s *consoleHandlerState) writeElement(str string, suffix *byte) {
	if s.Err != nil {
		return
	}

	if len(str) > 0 {
		if _, err := s.buffer.WriteString(str); err != nil {
			s.Err = err
			return
		}
	}

	if suffix != nil {
		if err := s.buffer.WriteByte(*suffix); err != nil {
			s.Err = err
		}
	}
}

func groupValues(value slog.Value) []string {
	value = value.Resolve()
	if value.Kind() != slog.KindGroup {
		return []string{value.String()}
	}

	var groupValue []string
	for _, v := range value.Group() {
		groupValue = append(groupValue, groupValues(v.Value)...)
	}
	return groupValue
}

package counter

import (
	"io"

	"github.com/qtraffics/qtfra/enhancements/iolib/underlay"
)

type Func func(n int64)

type ReadCounter interface {
	ReadCounters() []Func
}

type WriteCounter interface {
	WriteCounters() []Func
}

func UnwrapReadCounter(current io.Reader) (io.Reader, []Func) {
	var ans []Func
	for {
		if rc, ok := current.(ReadCounter); ok {
			ans = append(ans, rc.ReadCounters()...)

			if ur, ok := current.(underlay.Reader); ok {
				current = ur.UnderlayReader()
				continue
			}
		}
		return current, ans
	}
}

func UnwrapWriterCounter(current io.Writer) (io.Writer, []Func) {
	var ans []Func
	for {
		if rc, ok := current.(WriteCounter); ok {
			ans = append(ans, rc.WriteCounters()...)

			if uw, ok := current.(underlay.Writer); ok {
				current = uw.UnderlayWriter()
			}
			continue
		}
		return current, ans
	}
}

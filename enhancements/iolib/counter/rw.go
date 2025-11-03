package counter

import (
	"io"

	"github.com/qtraffics/qtfra/enhancements/iolib/underlay"
)

var (
	_ underlay.Writer = (*Writer)(nil)
	_ io.Writer       = (*Writer)(nil)
	_ WriteCounter    = (*Writer)(nil)
)

type Writer struct {
	w        io.Writer
	counters []Func
}

func NewWriter(w io.Writer, counters []Func) io.Writer {
	if w == nil || len(counters) == 0 {
		return w
	}
	// fast path
	if countWriter, isCountWriter := w.(*Writer); isCountWriter {
		countWriter.counters = append(countWriter.counters, counters...)
		return countWriter
	}

	// slow path
	var extraCounters []Func
	w, extraCounters = UnwrapWriterCounter(w)
	if len(extraCounters) > 0 {
		// keep the counters order
		counters = append(extraCounters, counters...)
	}

	return &Writer{w: w, counters: counters}
}

func (w *Writer) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	if n > 0 {
		for _, c := range w.counters {
			c(int64(n))
		}
	}
	return n, err
}

func (w *Writer) WriteCounters() []Func {
	return w.counters
}

func (w *Writer) UnderlayWriter() io.Writer {
	return w.w
}

var (
	_ underlay.Reader = (*Reader)(nil)
	_ io.Reader       = (*Reader)(nil)
	_ ReadCounter     = (*Reader)(nil)
)

type Reader struct {
	r       io.Reader
	counter []Func
}

func NewReader(r io.Reader, counters []Func) io.Reader {
	if r == nil || len(counters) == 0 {
		return r
	}
	// fast path
	if countReader, isCountReader := r.(*Reader); isCountReader {
		countReader.counter = append(countReader.counter, counters...)
		return countReader
	}

	// slow path
	var extraCounters []Func
	r, extraCounters = UnwrapReadCounter(r)
	if len(extraCounters) > 0 {
		// keep the counters order
		counters = append(extraCounters, counters...)
	}
	return &Reader{r: r, counter: counters}
}

func (r *Reader) ReadCounters() []Func {
	return r.counter
}

func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	if n > 0 {
		for _, c := range r.counter {
			c(int64(n))
		}
	}
	return n, err
}

func (r *Reader) UnderlayReader() io.Reader {
	return r.r
}

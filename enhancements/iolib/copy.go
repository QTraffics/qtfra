package iolib

import (
	"io"

	"github.com/QTraffics/qtfra/buf"
	"github.com/QTraffics/qtfra/ex"
)

// Deprecated: use BufWriter instead
func WriteBatch(w io.Writer, bs ...[]byte) (n int, err error) {
	if len(bs) == 0 {
		return 0, nil
	}
	if buffer, isBuffer := w.(*buf.Buffer); isBuffer {
		for _, b := range bs {
			nn, wErr := buffer.Write(b)
			n += nn
			if wErr != nil {
				err = wErr
				break
			}
		}
		return
	}

	buffer := buf.NewSize(16384)
	defer buffer.Free()

	for i := 0; i < len(bs); {
		current := bs[i]
		if len(current) == 0 {
			i++
			continue
		}

		offset := 0

		for offset < len(current) {
			// merge
			nn, bufferErr := buffer.Write(current[offset:])
			n += nn
			offset += nn

			if bufferErr != nil {
				if !ex.IsMulti(bufferErr, io.ErrShortBuffer) {
					err = bufferErr
					return
				}
				// buffer full , flush
				_, writeErr := buffer.WriteTo(w)
				if writeErr != nil {
					err = writeErr
					return
				}
				buffer.Reset()
			}
		}
		// next
		i++
	}

	if buffer.Len() > 0 {
		_, err = buffer.WriteTo(w)
	}

	return
}

package iolib

import (
	"fmt"
	"io"
	"syscall"

	"github.com/qtraffics/qtfra/buf"
	"github.com/qtraffics/qtfra/enhancements/iolib/counter"
	"github.com/qtraffics/qtfra/ex"
	"github.com/qtraffics/qtfra/sys/sysvars"
)

func Copy(source io.Reader, destination io.Writer) (n int64, err error) {
	var (
		readCounter  []counter.Func
		writeCounter []counter.Func
	)

	source, readCounter = counter.UnwrapReadCounter(source)
	destination, writeCounter = counter.UnwrapWriterCounter(destination)

	return CopyCounters(source, destination, readCounter, writeCounter)
}

func CopyCounters(source io.Reader, destination io.Writer, readCounters []counter.Func, writeCounters []counter.Func) (n int64, err error) {
	var (
		earlyCopied int
		pureCopied  int64
	)
	earlyCopied, source, err = copyEarly(source, destination, readCounters, writeCounters)
	n += int64(earlyCopied)
	if err != nil {
		return n, ex.Cause(err, "copyEarly")
	}

	pureCopied, err = copyPure(source, destination, readCounters, writeCounters)
	n += pureCopied
	if err != nil && !ex.IsMulti(err, io.EOF) {
		return n, ex.Cause(err, "copyPure")
	}

	return n, nil
}

func copyPure(source io.Reader, destination io.Writer, readCounter []counter.Func, writeCounter []counter.Func) (n int64, err error) {
	sourceSysConn, sourceIsSysConn := source.(syscall.Conn)
	destinationSysConn, destinationIsSysConn := destination.(syscall.Conn)
	if sourceIsSysConn && destinationIsSysConn {
		var (
			internalErr error

			sourceRawConn      syscall.RawConn
			destinationRawConn syscall.RawConn
		)

		if sourceRawConn, internalErr = sourceSysConn.SyscallConn(); internalErr != nil {
			goto genericCopy
		}
		if destinationRawConn, internalErr = destinationSysConn.SyscallConn(); internalErr != nil {
			goto genericCopy
		}
		var spliceHanded bool
		spliceHanded, n, err = splice(sourceRawConn, destinationRawConn, readCounter, writeCounter)
		if spliceHanded {
			return n, err
		}
	}

genericCopy:
	return copyGeneric(source, destination, readCounter, writeCounter)
}

func copyEarly(source io.Reader, destination io.Writer, readCounter []counter.Func, writeCounter []counter.Func) (n int, _ io.Reader, err error) {
	if needHandshake, ok := destination.(NeedHandshake); ok && needHandshake.NeedHandshake() {
		bufferHandshaker, isBufferHandshaker := destination.(HandshakeBuffer)
		if !isBufferHandshaker {
			err = destination.(Handshaker).Handshake()
			if err != nil {
				return n, source, ex.Cause(err, "handshake")
			}

		} else {
			var (
				handshakeBuffer *buf.Buffer
				handshakeErr    error
				handshakeWriteN int
			)

			source, handshakeBuffer = PickReaderCache(source)
			if handshakeBuffer == nil || handshakeBuffer.Empty() {
				handshakeBuffer = buf.New()
				handshakeReadN, handshakeReadErr := handshakeBuffer.ReadFromOnce(source)
				if handshakeReadErr != nil {
					return n, source, ex.Cause(handshakeReadErr, "handshakeRead")
				}

				if sysvars.DebugEnabled && handshakeReadN <= 0 {
					panic(fmt.Sprintf("handshake read zero or negative byte count from reader: %d", handshakeReadN))
				}
			}

			handshakeWriteN, handshakeErr = bufferHandshaker.Handshake(handshakeBuffer.Bytes())
			n += handshakeWriteN

			for _, c := range readCounter {
				c(int64(handshakeWriteN))
			}

			for _, c := range writeCounter {
				c(int64(handshakeWriteN))
			}

			if handshakeErr != nil {
				return n, source, ex.Cause(handshakeErr, "handshake")
			}
			_, _ = handshakeBuffer.Discard(handshakeWriteN)
			if handshakeWriteN < handshakeBuffer.Len() || !handshakeBuffer.Empty() {
				source = NewCacheReader(source, handshakeBuffer)
			} else {
				handshakeBuffer.Free()
			}
		}
	}

	for {
		var (
			earlyCopied int
			buffer      *buf.Buffer
		)

		source, buffer = PickReaderCache(source)
		if buffer == nil {
			break
		}

		earlyCopied, source, err = copyEarly(source, destination, readCounter, writeCounter)
		n += earlyCopied
		if err != nil {
			// Note: do not wrap this error here, may cause nested error message.
			return n, source, err
		}

		to, err := buffer.WriteTo(destination)
		n += int(to)
		if err != nil {
			return n, source, ex.Cause(err, "writeCache")
		}

		for _, c := range readCounter {
			c(to)
		}
		for _, c := range writeCounter {
			c(to)
		}

		if sysvars.DebugEnabled && !buffer.Empty() {
			panic("buffer not WriteTo fully")
		}
		buffer.Free()
	}
	return n, source, nil
}

func copyGeneric(source io.Reader, destination io.Writer, readCounter []counter.Func, writeCounter []counter.Func) (n int64, err error) {
	buffer := buf.NewHuge()
	defer buffer.Free()

	source = counter.NewReader(source, readCounter)
	destination = counter.NewWriter(destination, writeCounter)

	return io.CopyBuffer(destination, source, buffer.FreeBytes())
}

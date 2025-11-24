package iolib

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"testing"

	"github.com/qtraffics/qtfra/buf"
	"github.com/qtraffics/qtfra/enhancements/iolib/counter"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deployTCPWriteServer() (address net.Listener, err error) {
	listen, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	handleConn := func(conn net.Conn) {
		defer conn.Close()
		var needBytes uint64
		err = binary.Read(conn, binary.BigEndian, &needBytes)
		if err != nil {
			fmt.Println("server error: ", err)
			return
		}
		const maxDataSize = 1024 * 1024 * 1024 * 16 // 16G
		needBytes = min(needBytes, maxDataSize)
		hugeBuffer := buf.NewHuge()
		defer hugeBuffer.Free()
		for needBytes > 0 {
			needWritten := min(needBytes, uint64(hugeBuffer.Size()))
			data := hugeBuffer.FreeBytes()[:needWritten]
			n, err := conn.Write(data)
			if err != nil {
				fmt.Println("server error: ", err)
				return
			}
			needBytes -= uint64(n)
		}
	}

	go func() {
		for {
			conn, err := listen.Accept()
			if err != nil || conn == nil {
				break
			}
			go handleConn(conn)
		}
	}()

	return listen, nil
}

func deployTCPReadServer() (address net.Listener, err error) {
	listen, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	handleConn := func(conn net.Conn) {
		defer conn.Close()
		var needBytes uint64
		err = binary.Read(conn, binary.BigEndian, &needBytes)
		if err != nil {
			fmt.Println("server error: ", err)
			return
		}
		const maxDataSize = 1024 * 1024 * 1024 * 16 // 16G
		needBytes = min(needBytes, maxDataSize)
		hugeBuffer := buf.NewHuge()
		defer hugeBuffer.Free()
		for needBytes > 0 {
			needRead := min(needBytes, uint64(hugeBuffer.Size()))
			data := hugeBuffer.FreeBytes()[:needRead]
			n, err := conn.Read(data)
			if err != nil {
				fmt.Println("server error: ", err)
				return
			}
			needBytes -= uint64(n)
		}
	}

	go func() {
		for {
			conn, err := listen.Accept()
			if err != nil || conn == nil {
				break
			}
			go handleConn(conn)
		}
	}()

	return listen, nil
}

type noSplice struct {
	r io.Reader
	w io.Writer
}

func (r *noSplice) Write(p []byte) (n int, err error) {
	return r.w.Write(p)
}

func (r *noSplice) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

type orderedReader struct {
	n uint8
}

func (r *orderedReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = r.n
		r.n++
	}
	return len(p), nil
}

func (r *orderedReader) Max() int {
	return math.MaxUint8
}

func TestCopyFeature(t *testing.T) {
	const size int64 = 4095
	t.Run("with counter", func(t *testing.T) {
		var readCountN int64
		var writeCountN int64

		var (
			readCounter = []counter.Func{func(n int64) {
				readCountN += n
			}}
			writeCounter = []counter.Func{func(n int64) {
				writeCountN += n
			}}
		)
		source := counter.NewReader(io.LimitReader(new(orderedReader), size), readCounter)
		destination := counter.NewWriter(io.Discard, writeCounter)

		n, err := Copy(destination, source)
		require.Nil(t, err)
		assert.Equalf(t, size, n, "size != n; n=%d", n)
	})

	t.Run("with buffer", func(t *testing.T) {
		buffer := buf.NewMinimal()

		_, _ = buffer.ReadFromOnce(new(orderedReader)) // fill buffer
		var (
			sourceReader      = NewCacheReader(io.LimitReader(new(orderedReader), size), buffer)
			destinationWriter = buf.NewSize(buffer.Len() + int(size))
		)
		defer buffer.Free()

		n, err := Copy(destinationWriter, sourceReader)
		require.Nil(t, err)
		exceptSize := size + int64(buffer.Size())
		assert.Equalf(t, exceptSize, n, "exceptSize != n; n=%d", n)
	})

	t.Run("with buffers", func(t *testing.T) {
		var (
			buffers    []*buf.Buffer
			dataSource = new(orderedReader)

			readded int64
		)
		defer func() {
			for len(buffers) > 0 {
				buffers[0].Free()
				buffers[0] = nil
				buffers = buffers[1:]
			}
		}()

		for range 3 {
			buffer := buf.NewMinimal()
			n, err := buffer.ReadFromOnce(dataSource)
			assert.Nil(t, err)
			assert.Equal(t, buffer.Size(), n)
			buffers = append(buffers, buffer)
			readded += int64(n)
		}
		r := NewCacheReaderList(io.LimitReader(dataSource, int64(dataSource.Max())), buffers)
		readded += int64(dataSource.Max())
		writerBuffer := buf.NewSize(int(readded))
		defer writerBuffer.Free()
		originalData, _ := io.ReadAll(io.LimitReader(new(orderedReader), int64(readded)))

		nn, err := Copy(writerBuffer, r)
		require.Nil(t, err)
		assert.Equal(t, readded, nn)
		assert.Equal(t, originalData, writerBuffer.Bytes())
	})
}

func TestCopyConn(t *testing.T) {
	const hugeSize int64 = 1024 * 1024 * 1024

	writeAddress, err := deployTCPWriteServer()
	assert.Nil(t, err, "deployTCPWriteServer")
	defer writeAddress.Close()
	readAddress, err := deployTCPReadServer()
	assert.Nil(t, err, "deployTCPReadServer")
	defer readAddress.Close()
	createConn := func(size uint64) (source net.Conn, destination net.Conn, err error) {
		sourceConn, err := net.Dial("tcp", writeAddress.Addr().String())
		if err != nil {
			return nil, nil, err
		}
		err = binary.Write(sourceConn, binary.BigEndian, size)
		if err != nil {
			return nil, nil, err
		}

		destinationConn, err := net.Dial("tcp", readAddress.Addr().String())
		if err != nil {
			return nil, nil, err
		}
		err = binary.Write(destinationConn, binary.BigEndian, size)
		if err != nil {
			return nil, nil, err
		}

		return sourceConn, destinationConn, nil
	}

	t.Run("no counter", func(t *testing.T) {
		source, destination, err := createConn(uint64(hugeSize))
		require.Nil(t, err, "createConn")
		defer source.Close()
		defer destination.Close()
		testSpliceTriggered = false
		n, err := Copy(destination, source)
		require.Nil(t, err)

		assert.Equalf(t, hugeSize, n, "hugeSize != n; n=%d", n)
		require.True(t, testSpliceTriggered)
	})

	t.Run("with counter", func(t *testing.T) {
		source, destination, err := createConn(uint64(hugeSize))
		assert.Nil(t, err, "createConn")
		defer source.Close()
		defer destination.Close()
		var (
			readCountN  int64
			writeCountN int64
		)

		sourceReader := counter.NewReader(source, []counter.Func{func(n int64) {
			readCountN += n
		}})
		destinationWriter := counter.NewWriter(destination, []counter.Func{func(n int64) {
			writeCountN += n
		}})
		testSpliceTriggered = false
		n, err := Copy(destinationWriter, sourceReader)

		require.Nil(t, err, "Copy")
		require.True(t, testSpliceTriggered)
		assert.Equalf(t, hugeSize, n, "hugeSize != n; n=%d", n)
		assert.Equalf(t, n, readCountN, "n != readCountN; readCountN=%d", n)
		assert.Equalf(t, n, writeCountN, "n != writeCountN; writeCountN=%d", n)
	})

	t.Run("with counter no splice", func(t *testing.T) {
		source, destination, err := createConn(uint64(hugeSize))
		assert.Nil(t, err, "createConn")
		defer source.Close()
		defer destination.Close()
		var (
			readCountN  int64
			writeCountN int64

			sourceReader      io.Reader = &noSplice{r: source}
			destinationWriter io.Writer = &noSplice{w: destination}
		)

		sourceReader = counter.NewReader(sourceReader, []counter.Func{func(n int64) {
			readCountN += n
		}})
		destinationWriter = counter.NewWriter(destinationWriter, []counter.Func{func(n int64) {
			writeCountN += n
		}})
		testSpliceTriggered = false
		n, err := Copy(destinationWriter, sourceReader)

		require.Nil(t, err, "Copy")
		require.False(t, testSpliceTriggered)

		assert.Equalf(t, hugeSize, n, "hugeSize != n; n=%d", n)
		assert.Equalf(t, n, readCountN, "n != readCountN; readCountN=%d", n)
		assert.Equalf(t, n, writeCountN, "n != writeCountN; writeCountN=%d", n)
	})
}

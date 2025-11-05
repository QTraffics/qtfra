package iolib

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"testing"

	"github.com/qtraffics/qtfra/buf"
	"github.com/qtraffics/qtfra/enhancements/iolib/counter"

	"github.com/stretchr/testify/assert"
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

type zeroReader struct{}

func (r *zeroReader) Read(p []byte) (int, error) {
	return len(p), nil
}

type noSpliceReader struct {
	r io.Reader
}

func (r *noSpliceReader) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func TestCopyConn(t *testing.T) {
	const testSize int64 = 1024 * 1024 * 1024

	t.Run("copy conn", func(t *testing.T) {
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
			source, destination, err := createConn(uint64(testSize))
			assert.Nil(t, err, "createConn")
			defer source.Close()
			defer destination.Close()

			n, err := Copy(source, destination)
			assert.Nil(t, err)
			assert.Equalf(t, testSize, n, "testSize != n; n=%d", n)
		})

		t.Run("with counter", func(t *testing.T) {
			source, destination, err := createConn(uint64(testSize))
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
			n, err := Copy(sourceReader, destinationWriter)

			assert.Nil(t, err, "Copy")
			assert.Equalf(t, testSize, n, "testSize != n; n=%d", n)
			assert.Equalf(t, n, readCountN, "n != readCountN; readCountN=%d", n)
			assert.Equalf(t, n, writeCountN, "n != writeCountN; writeCountN=%d", n)
		})

		t.Run("with counter no splice", func(t *testing.T) {
			source, destination, err := createConn(uint64(testSize))
			assert.Nil(t, err, "createConn")
			defer source.Close()
			defer destination.Close()
			var (
				readCountN  int64
				writeCountN int64

				sourceReader io.Reader = &noSpliceReader{r: source}
			)

			sourceReader = counter.NewReader(sourceReader, []counter.Func{func(n int64) {
				readCountN += n
			}})
			destinationWriter := counter.NewWriter(destination, []counter.Func{func(n int64) {
				writeCountN += n
			}})
			n, err := Copy(sourceReader, destinationWriter)

			assert.Nil(t, err, "Copy")
			assert.Equalf(t, testSize, n, "testSize != n; n=%d", n)
			assert.Equalf(t, n, readCountN, "n != readCountN; readCountN=%d", n)
			assert.Equalf(t, n, writeCountN, "n != writeCountN; writeCountN=%d", n)
		})
	})

	t.Run("copy common", func(t *testing.T) {
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
			source := counter.NewReader(io.LimitReader(&zeroReader{}, testSize), readCounter)
			destination := counter.NewWriter(io.Discard, writeCounter)

			n, err := Copy(source, destination)
			assert.Nil(t, err)
			assert.Equalf(t, testSize, n, "testSize != n; n=%d", n)
		})

		t.Run("with buffer", func(t *testing.T) {
			var (
				readCountN  int64
				writeCountN int64
				buffer      = buf.NewHuge()
			)

			_, _ = buffer.ReadFromOnce(&zeroReader{}) // fill buffer
			var (
				readCounter = []counter.Func{func(n int64) {
					readCountN += n
				}}
				writeCounter = []counter.Func{func(n int64) {
					writeCountN += n
				}}

				sourceReader      = NewCacheReader(io.LimitReader(&zeroReader{}, testSize), buffer)
				destinationWriter = io.Discard
			)
			defer buffer.Free()

			source := counter.NewReader(sourceReader, readCounter)
			destination := counter.NewWriter(destinationWriter, writeCounter)

			n, err := Copy(source, destination)
			assert.Nil(t, err)
			exceptSize := testSize + int64(buffer.Size())
			assert.Equalf(t, exceptSize, n, "exceptSize != n; n=%d", n)
			assert.Equalf(t, exceptSize, readCountN, "exceptSize != readCountN; readCountN=%d", readCountN)
			assert.Equalf(t, exceptSize, writeCountN, "exceptSize != writeCountN; writeCountN=%d", writeCountN)
		})
	})
}

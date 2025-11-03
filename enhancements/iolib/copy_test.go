package iolib

import (
	"encoding/binary"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/qtraffics/qtfra/buf"
)

func deployTCPWriteServer() (address string, err error) {
	listen, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
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
			if err != nil {
				panic(err)
			}
			go handleConn(conn)
		}
	}()

	return listen.Addr().String(), nil
}

func deployTCPReadServer() (address string, err error) {
	listen, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
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
			if err != nil {
				panic(err)
			}
			go handleConn(conn)
		}
	}()

	return listen.Addr().String(), nil
}

func TestCopyConn(t *testing.T) {
	writeAddress, err := deployTCPWriteServer()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	readAddress, err := deployTCPReadServer()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	createConn := func(t *testing.T, size uint64) (source net.Conn, destination net.Conn, err error) {
		sourceConn, err := net.Dial("tcp", writeAddress)
		if err != nil {
			return nil, nil, err
		}
		err = binary.Write(sourceConn, binary.BigEndian, size)
		if err != nil {
			return nil, nil, err
		}
		destinationConn, err := net.Dial("tcp", readAddress)
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
		const size uint64 = 1024 * 1024 * 1024 * 4 - 1
		source, destination, err := createConn(t, size)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		start := time.Now()
		n, err := Copy(source, destination)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		if uint64(n) != size {
			t.Errorf("n != size, n=%d", n)
			t.FailNow()
		}
		fmt.Println("Consume: ", time.Now().Sub(start))
	})
}

func TestCopyFile(t *testing.T) {
}

package iolib

import (
	"crypto/rand"
	"io"
	"testing"

	"github.com/qtraffics/qtfra/buf"
	"github.com/qtraffics/qtfra/ex"

	"github.com/stretchr/testify/assert"
)

func TestCacheReader(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		originalReader := buf.NewMinimal()
		fillRandom(originalReader)
		originalData := originalReader.Bytes()
		originalLen := originalReader.Len()
		defer originalReader.Free()

		buf1 := buf.NewSize(originalLen / 2)
		defer buf1.Free()
		_, err := buf1.ReadFromOnce(originalReader)
		assert.Nil(t, err)

		reader := NewCacheReader(originalReader, buf1)
		readBuffer := make([]byte, originalLen)
		var n int
		n, err = reader.Read(readBuffer[:])
		assert.Nil(t, err)
		assert.Equal(t, originalLen, n)
		assert.Equal(t, originalData, readBuffer)
	})

	t.Run("multi", func(t *testing.T) {
		originalReader := buf.NewMinimal()
		defer originalReader.Free()
		fillRandom(originalReader)
		originalData := originalReader.Bytes()
		originalLen := originalReader.Len()

		var buffers []*buf.Buffer
		const splits = 4
		for range splits - 1 {
			size := int(originalLen / splits)
			buffer := buf.NewSize(size)
			nn, err := buffer.ReadFromOnce(originalReader)
			assert.Nil(t, err)
			assert.Equal(t, size, nn)

			buffers = append(buffers, buffer)
		}
		r := NewCacheReaderList(originalReader, buffers)
		all, err := io.ReadAll(r)
		assert.Nil(t, err)
		assert.Equal(t, originalLen, len(all))
		assert.Equal(t, originalData, all)
	})
}

func fillRandom(b *buf.Buffer) {
	_, err := b.ReadFrom(rand.Reader)
	if err != nil && !ex.IsMulti(err, io.ErrShortBuffer) {
		panic(err)
	}
}

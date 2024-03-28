package encoding

import (
	"fmt"
	"io"
	"sync"
)

type Buffer interface {
	Data() []byte
	SetData([]byte)
	Free()
}

// TODO: Move grpc.SharedBufferPool to separate package to make it importable
// without introducing import cycle.
var globalBufferPool = sync.Pool{New: func() any { return []byte(nil) }}

type buffer struct {
	data []byte
}

func (b *buffer) Data() []byte {
	return b.data
}

func (b *buffer) SetData(data []byte) {
	b.data = data
}

func (b *buffer) Free() {
	globalBufferPool.Put(ClearBuffer(b.data))
}

func NewBuffer(size int) Buffer {
	data := globalBufferPool.Get().([]byte)
	if cap(data) < size {
		if cap(data) > 0 {
			globalBufferPool.Put(data)
		}
		data = make([]byte, size)
	} else {
		data = data[:size]
	}
	return &buffer{data: data}
}

func ClearBuffer(buf []byte) []byte {
	// TODO: replace with clear when go1.21 is supported: clear(buf)
	for i := range buf {
		buf[i] = 0
	}
	buf = buf[:0]
	return buf
}

// BufferSeq is the equivalent of [iter.Seq][[Buffer], error], but cannot be added by
// directly referencing the new [iter] package since it is not yet supported in
// all versions of go supported by grpc-go.
type BufferSeq = func(yield func(Buffer, error) bool)

func ErrBufferSeq(err error) BufferSeq {
	return OneElementSeq(nil, err)
}

func OneElementSeq(buf Buffer, err error) BufferSeq {
	return func(yield func(Buffer, error) bool) {
		yield(buf, err)
	}
}

func FullRead(expectedLength int, data BufferSeq, newBuffer func(int) Buffer) (buf Buffer, err error) {
	var buffers []Buffer
	var receivedSize int
	defer func() {
		for _, b := range buffers {
			b.Free()
		}
	}()

	data(func(buf Buffer, innerErr error) bool {
		if innerErr != nil {
			err = innerErr
			return false
		}

		buffers = append(buffers, buf)
		receivedSize += len(buf.Data())
		return true
	})

	if err != nil {
		return nil, err
	}

	if receivedSize != expectedLength {
		return nil, fmt.Errorf("proto: did not receive expected data size %d, got %d (%w)",
			expectedLength, receivedSize, io.ErrShortBuffer)
	}

	var fullBuffer Buffer

	// If the entire data was received in one buffer, avoid copying altogether and use that one directly
	if len(buffers[0].Data()) == expectedLength {
		fullBuffer = buffers[0]
		// Prevent the defer from freeing the buffer
		buffers = buffers[1:]
	} else {
		// Otherwise, materialize the buffer
		fullBuffer = newBuffer(receivedSize)
		fullBuffer.SetData(fullBuffer.Data()[:0])
		for _, b := range buffers {
			fullBuffer.SetData(append(fullBuffer.Data(), b.Data()...))
		}
	}

	return fullBuffer, nil
}

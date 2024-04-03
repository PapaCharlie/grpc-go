package encoding

import (
	internalencoding "google.golang.org/grpc/internal"
)

type Buffer interface {
	Data() []byte
	Free()
}

// TODO(PapaCharlie): Move grpc.SharedBufferPool to separate package to make it
// importable without introducing import cycle.
var globalBufferPool = internalencoding.NewSimpleSharedBufferPool()

type buffer struct {
	data []byte
}

func (b *buffer) Data() []byte {
	return b.data
}

func (b *buffer) Free() {
	if b.data != nil {
		ClearBuffer(b.data)
		globalBufferPool.Put(&b.data)
		b.data = nil
	}
}

func NewBuffer(size int) Buffer {
	return &buffer{data: globalBufferPool.Get(size)}
}

func ClearBuffer(buf []byte) {
	// TODO: replace with clear when go1.21 is supported: clear(buf)
	for i := range buf {
		buf[i] = 0
	}
}

type simpleBuffer struct {
	data []byte
}

func (s *simpleBuffer) Data() []byte {
	return s.data
}

func (s *simpleBuffer) Free() {}

func SimpleBuffer(data []byte) Buffer {
	return &simpleBuffer{data}
}

// BufferSeq is the equivalent of [iter.Seq][[Buffer]], but cannot be added by
// directly referencing the new [iter] package since it is not yet supported in
// all versions of go supported by grpc-go.
type BufferSeq []Buffer

func (s BufferSeq) Size() (i int) {
	for _, b := range s {
		i += len(b.Data())
	}
	return i
}

func (s BufferSeq) Free() {
	for _, b := range s {
		b.Free()
	}
}

func (s BufferSeq) Concat(provider BufferProvider) Buffer {
	// If the entire data was received in one buffer, avoid copying altogether and use that one directly
	if len(s) == 1 {
		return s[0]
	} else {
		// Otherwise, materialize the buffer
		buf := provider(s.Size())
		idx := 0
		for _, b := range s {
			idx += copy(buf.Data()[idx:], b.Data())
			b.Free()
		}

		return buf
	}
}

type BufferProvider = func(int) Buffer

package grpc

import (
	"sync/atomic"

	"google.golang.org/grpc/encoding"
)

type buffer struct {
	data     []byte
	refCount atomic.Int32
	source   encoding.BufferProvider
}

func (b *buffer) free() {
	if b.refCount.Add(-1) == 0 {
		b.source.ReturnBuffer(b.data)
		b.data = nil
	}
}

func (b *buffer) incRefCount() {
	b.refCount.Add(1)
}

func newBuffer(data []byte, source encoding.BufferProvider) *buffer {
	b := &buffer{
		data:   data,
		source: source,
	}
	b.incRefCount()
	return b
}

func freeAll(buffers []*buffer) {
	for _, b := range buffers {
		b.free()
	}
}

func bufferSliceSize(buffers []*buffer) (size int) {
	for _, b := range buffers {
		size += len(b.data)
	}
	return size
}

func referenceAll(buffers []*buffer) {
	for _, b := range buffers {
		b.incRefCount()
	}
}

func unwrapBufferSlice(buffers []*buffer) [][]byte {
	out := make([][]byte, len(buffers))
	for i, b := range buffers {
		out[i] = b.data
	}
	return out
}

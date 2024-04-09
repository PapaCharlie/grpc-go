package internal

import (
	"sync/atomic"

	"google.golang.org/grpc/bufslice"
)

func NewRefCountedBufSlice(data [][]byte, provider bufslice.BufferProvider) *RefCountedBufSlice {
	ref := &RefCountedBufSlice{
		data:     data,
		provider: provider,
	}
	return ref.Ref()
}

type RefCountedBufSlice struct {
	data     [][]byte
	provider bufslice.BufferProvider
	count    atomic.Int32
}

func (ref *RefCountedBufSlice) Data() [][]byte {
	return ref.data
}

func (ref *RefCountedBufSlice) Len() int {
	return bufslice.Len(ref.data)
}

func (ref *RefCountedBufSlice) Reader() *bufslice.Reader {
	return bufslice.NewReader(ref.data)
}

func (ref *RefCountedBufSlice) Writer() *bufslice.Writer {
	return bufslice.NewWriter(&ref.data, ref.provider)
}

func (ref *RefCountedBufSlice) Free() {
	if ref.count.Add(-1) == 0 {
		bufslice.ReturnAll(ref.data, ref.provider)
	}
}

func (ref *RefCountedBufSlice) Ref() *RefCountedBufSlice {
	ref.count.Add(1)
	return ref
}

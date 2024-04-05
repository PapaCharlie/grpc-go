package encoding

import (
	"io"
)

func ClearBuffer(buf []byte) {
	// TODO: replace with clear when go1.21 is supported: clear(buf)
	for i := range buf {
		buf[i] = 0
	}
}

type BufferProvider interface {
	GetBuffer(size int) []byte
	ReturnBuffer([]byte)
}

type sliceWriter struct {
	buffers  *[][]byte
	provider BufferProvider
}

func (s *sliceWriter) appendNewBuffer(size int) []byte {
	buf := newBuffer(size, s.provider)
	*s.buffers = append(*s.buffers, buf)
	return buf
}

func (s *sliceWriter) Write(p []byte) (n int, err error) {
	n = len(p)

	lastIdx := len(*s.buffers) - 1
	var lastBuffer []byte
	if lastIdx != -1 {
		lastBuffer = (*s.buffers)[lastIdx]
	}

	if lastIdx == -1 || cap(lastBuffer) == len(lastBuffer) {
		lastBuffer = s.appendNewBuffer(len(p))
		lastIdx++
	}

	if availableCapacity := cap(lastBuffer) - len(lastBuffer); availableCapacity < len(p) {
		p = p[copy(lastBuffer[len(lastBuffer):cap(lastBuffer)], p):]
		(*s.buffers)[lastIdx] = lastBuffer[:cap(lastBuffer)]

		lastBuffer = s.appendNewBuffer(len(p) - availableCapacity)
		lastIdx++
	}

	(*s.buffers)[lastIdx] = append(lastBuffer, p...)

	return n, nil
}

func BufferSliceWriter(buffers *[][]byte, provider BufferProvider) io.Writer {
	return &sliceWriter{buffers: buffers, provider: provider}
}

func BufferSliceSize(buffers [][]byte) (l int) {
	for _, b := range buffers {
		l += len(b)
	}
	return l
}

func ConcatBufferSlice(buffers [][]byte, provider BufferProvider, alwaysCopy bool) []byte {
	// If the entire data was received in one buffer, avoid copying altogether and use that one directly
	if len(buffers) == 1 && !alwaysCopy {
		return buffers[0]
	} else {
		// Otherwise, materialize the buffer
		buf := newBuffer(BufferSliceSize(buffers), provider)
		if provider == nil {
			buf = make([]byte, BufferSliceSize(buffers))
		} else {
			buf = provider.GetBuffer(BufferSliceSize(buffers))
		}
		idx := 0
		for _, b := range buffers {
			idx += copy(buf[idx:], b)
		}

		return buf
	}
}

func newBuffer(size int, provider BufferProvider) []byte {
	if provider == nil {
		return make([]byte, size)
	} else {
		return provider.GetBuffer(size)
	}
}

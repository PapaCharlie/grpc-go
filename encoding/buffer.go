package encoding

import (
	"io"
)

type BufferProvider interface {
	GetBuffer(size int) []byte
	ReturnBuffer([]byte)
}

type NoopBufferProvider struct{}

func (n NoopBufferProvider) GetBuffer(size int) []byte {
	return make([]byte, size)
}

func (n NoopBufferProvider) ReturnBuffer(bytes []byte) {}

type sliceWriter struct {
	buffers  *[][]byte
	provider BufferProvider
}

func (s *sliceWriter) appendNewBuffer(size int) []byte {
	buf := s.provider.GetBuffer(size)
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

func NewBufferSliceWriter(buffers *[][]byte, provider BufferProvider) io.Writer {
	return &sliceWriter{buffers: buffers, provider: provider}
}

func BufferSliceSize(buffers [][]byte) (l int) {
	for _, b := range buffers {
		l += len(b)
	}
	return l
}

func WriteBufferSlice(buffers [][]byte, out []byte) {
	out = out[:0]
	for _, b := range buffers {
		out = append(out, b...)
	}
}

func MaterializeBufferSlice(buffers [][]byte) []byte {
	buf := make([]byte, 0, BufferSliceSize(buffers))
	WriteBufferSlice(buffers, buf)
	return buf
}

type BufferSliceReader struct {
	data     [][]byte
	len, idx int
}

func (r *BufferSliceReader) Len() int {
	return r.len - r.idx
}

func (r *BufferSliceReader) Read(buf []byte) (n int, _ error) {
	for len(buf) != 0 && r.len != 0 {
		data := r.data[0]
		copied := copy(buf, data[r.idx:])
		r.len -= copied

		buf = buf[copied:]

		if copied == len(data) {
			r.data = r.data[1:]
			r.idx = 0
		} else {
			r.idx += copied
		}
		n += copied
	}

	if n == 0 {
		return 0, io.EOF
	}

	return n, nil
}

func NewBufferSliceReader(data [][]byte) *BufferSliceReader {
	return &BufferSliceReader{
		data: data,
		len:  BufferSliceSize(data),
	}
}

func ReturnAllBuffers(data [][]byte, provider BufferProvider) {
	for _, b := range data {
		provider.ReturnBuffer(b)
	}
}

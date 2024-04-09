package bufslice

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

type Writer struct {
	buffers  *[][]byte
	provider BufferProvider
}

func (s *Writer) appendNewBuffer(size int) []byte {
	buf := s.provider.GetBuffer(size)[:0]
	*s.buffers = append(*s.buffers, buf)
	return buf
}

func (s *Writer) Write(p []byte) (n int, err error) {
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
		(*s.buffers)[lastIdx] = append(lastBuffer, p[:availableCapacity]...)
		p = p[availableCapacity:]

		lastBuffer = s.appendNewBuffer(len(p) - availableCapacity)
		lastIdx++
	}

	(*s.buffers)[lastIdx] = append(lastBuffer, p...)

	return n, nil
}

func (s *Writer) ReadFrom(r io.Reader) (n int64, err error) {
	// TODO(PapaCharlie): default to the max http2 frame size used by the underlying
	// http/2 transport, however this can likely be improved. Maybe BufferProvider
	// can optionally implement an interface that hints at the optimal buffer size?
	const chunkSize = 16 * 1024

	for {
		buf := s.appendNewBuffer(chunkSize)
		// Always maximize how much of the buffer is reused if the provider returned a
		// larger buffer.
		buf = buf[:cap(buf)]
		read, err := r.Read(buf)
		buf = buf[:read]
		(*s.buffers)[len(*s.buffers)-1] = buf
		n += int64(read)
		if err != nil {
			return n, err
		}
	}
}

func NewWriter(buffers *[][]byte, provider BufferProvider) *Writer {
	return &Writer{buffers: buffers, provider: provider}
}

func Len(buffers [][]byte) (l int) {
	for _, b := range buffers {
		l += len(b)
	}
	return l
}

func WriteTo(buffers [][]byte, out []byte) {
	out = out[:0]
	for _, b := range buffers {
		out = append(out, b...)
	}
}

func Materialize(buffers [][]byte) []byte {
	buf := make([]byte, 0, Len(buffers))
	WriteTo(buffers, buf)
	return buf
}

type Reader struct {
	data              [][]byte
	len               int
	dataIdx, sliceIdx int
}

func (r *Reader) Data() [][]byte {
	return r.data
}

func (r *Reader) Len() int {
	return r.len
}

func (r *Reader) Read(buf []byte) (n int, _ error) {
	for len(buf) != 0 && r.len != 0 {
		data := r.data[r.dataIdx]
		copied := copy(buf, data[r.sliceIdx:])
		r.len -= copied

		buf = buf[copied:]

		if copied == len(data) {
			r.dataIdx++
			r.sliceIdx = 0
		} else {
			r.sliceIdx += copied
		}
		n += copied
	}

	if n == 0 {
		return 0, io.EOF
	}

	return n, nil
}

func NewReader(data [][]byte) *Reader {
	return &Reader{
		data: data,
		len:  Len(data),
	}
}

func ReturnAll(data [][]byte, provider BufferProvider) {
	for _, b := range data {
		provider.ReturnBuffer(b)
	}
}

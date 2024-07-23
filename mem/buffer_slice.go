package mem

import (
	"io"
)

// BufferSlice offers a means to represent data that spans one or more Buffer
// instances.
type BufferSlice []*Buffer

// Len returns the sum of the length of all the Buffers in this slice.
//
// # Warning
//
// Invoking the built-in len on a BufferSlice will return the number of buffers
// in the slice, and *not* the value returned by this function.
func (s BufferSlice) Len() int {
	var length int
	for _, b := range s {
		length += b.Len()
	}
	return length
}

// Ref returns a new BufferSlice containing a new reference of each Buffer in the
// input slice.
func (s BufferSlice) Ref() BufferSlice {
	out := make(BufferSlice, len(s))
	for i, b := range s {
		out[i] = b.Ref()
	}
	return out
}

// Free invokes Buffer.Free on each Buffer in the slice.
func (s BufferSlice) Free() {
	for _, b := range s {
		b.Free()
	}
}

// CopyTo copies each of the underlying Buffer's data into the given buffer,
// returning the number of bytes copied.
func (s BufferSlice) CopyTo(out []byte) int {
	off := 0
	for _, b := range s {
		off += copy(out[off:], b.ReadOnlyData())
	}
	return off
}

// Materialize concatenates all the underlying Buffer's data into a single
// contiguous buffer using CopyTo.
func (s BufferSlice) Materialize() []byte {
	l := s.Len()
	if l == 0 {
		return nil
	}
	out := make([]byte, l)
	s.CopyTo(out)
	return out
}

// LazyMaterialize functions like Materialize except that it writes the data to a
// single Buffer pulled from the given BufferPool. As a special case, if the
// input BufferSlice only actually has one Buffer, this function has nothing to
// do and simply returns said Buffer, hence it being "lazy".
func (s BufferSlice) LazyMaterialize(pool BufferPool) *Buffer {
	if len(s) == 1 {
		return s[0].Ref()
	}
	buf := pool.Get(s.Len())
	s.CopyTo(buf)
	return NewBuffer(buf, pool.Put)
}

// Reader returns a new Reader for the input slice after taking references to
// each underlying buffer.
func (s BufferSlice) Reader() *Reader {
	return &Reader{
		data: s.Ref(),
		len:  s.Len(),
	}
}

var _ io.ReadCloser = (*Reader)(nil)

// Reader exposes a BufferSlice's data as an io.Reader, allowing it to interface
// with other parts systems. It also provides an additional convenience method
// Remaining which returns the number of unread bytes remaining in the slice. It
// frees the underlying buffers as it finishes reading them.
type Reader struct {
	data BufferSlice
	len  int
	idx  int
}

// Remaining returns the number of unread bytes remaining in the slice.
func (r *Reader) Remaining() int {
	return r.len
}

// Close frees the underlying BufferSlice and never returns an error. Subsequent
// calls to Read will return (0, io.EOF).
func (r *Reader) Close() error {
	r.data.Free()
	r.data = nil
	r.len = 0
	return nil
}

func (r *Reader) Read(buf []byte) (n int, _ error) {
	for len(buf) != 0 && r.len != 0 {
		data := r.data[0].ReadOnlyData()
		copied := copy(buf, data[r.idx:])
		r.len -= copied

		buf = buf[copied:]

		if copied == len(data) {
			oldBuffer := r.data[0]
			oldBuffer.Free()
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

var _ io.Writer = (*writer)(nil)

type writer struct {
	buffers *BufferSlice
	pool    BufferPool
}

func (w *writer) Write(p []byte) (n int, err error) {
	b := Copy(p, w.pool)
	*w.buffers = append(*w.buffers, b)
	return b.Len(), nil
}

// NewWriter wraps the given BufferSlice and BufferPool to implement the
// io.Writer interface. Every call to Write copies the contents of the given
// buffer into a new Buffer pulled from the given pool and the Buffer is added to
// the given BufferSlice. For example, in the context of a http.Handler, the
// following code can be used to copy the contents of a request into a
// BufferSlice:
//
//	var out BufferSlice
//	n, err := io.Copy(mem.NewWriter(&out, pool), req.Body)
func NewWriter(buffers *BufferSlice, pool BufferPool) io.Writer {
	return &writer{buffers: buffers, pool: pool}
}
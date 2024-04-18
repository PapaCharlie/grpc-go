/*
 *
 * Copyright 2024 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

/*
Package mem does memory things
*/
package mem

import (
	"io"
	"sync/atomic"
)

type BufferSlice []*Buffer

type Buffer struct {
	data []byte
	refs *atomic.Int32
	free func([]byte)
}

func NewBuffer(data []byte, free func([]byte)) *Buffer {
	return (&Buffer{data: data, refs: new(atomic.Int32), free: free}).Ref()
}

func Copy(data []byte, pool BufferPool) *Buffer {
	buf := pool.Get(len(data))
	copy(buf, data)
	return NewBuffer(buf, pool.Put)
}

func (b *Buffer) ReadOnlyData() []byte {
	if b == nil || b.refs.Load() <= 0 {
		return nil
	}
	return b.data
}

func (b *Buffer) Ref() *Buffer {
	b.refs.Add(1)
	return b
}

func (b *Buffer) Free() {
	if b == nil {
		return
	}
	refs := b.refs.Add(-1)
	if refs != 0 {
		return
	}

	if b.free != nil {
		b.free(b.data)
	}
	b.data = nil
}

func (b *Buffer) Len() int {
	return len(b.ReadOnlyData())
}

func (b *Buffer) Split(n int) *Buffer {
	data := b.data
	free := b.free

	b.data = data[:n]
	b.free = nil

	newBuf := &Buffer{
		data: data[n:],
		refs: b.refs,
		free: func(_ []byte) {
			free(data)
		},
	}

	return newBuf.Ref()
}

type Writer struct {
	buffers *BufferSlice
	pool    BufferPool
}

func (s *Writer) Write(p []byte) (n int, err error) {
	buf := s.pool.Get(len(p))
	n = copy(buf, p)

	*s.buffers = append(*s.buffers, NewBuffer(buf, s.pool.Put))

	return n, nil
}

func NewWriter(buffers *BufferSlice, pool BufferPool) *Writer {
	return &Writer{buffers: buffers, pool: pool}
}

type Reader struct {
	data BufferSlice
	len  int
	idx  int
}

func (r *Reader) Len() int {
	return r.len
}

func (r *Reader) Read(buf []byte) (n int, _ error) {
	for len(buf) != 0 && r.len != 0 {
		data := r.data[0].ReadOnlyData()
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

func (s BufferSlice) Reader() *Reader {
	return &Reader{
		data: s,
		len:  s.Len(),
	}
}

func (s BufferSlice) Len() (length int) {
	for _, b := range s {
		length += len(b.ReadOnlyData())
	}
	return length
}

func (s BufferSlice) Ref() BufferSlice {
	for _, b := range s {
		b.Ref()
	}
	return s
}

func (s BufferSlice) Free() {
	for _, b := range s {
		b.Free()
	}
}

func (s BufferSlice) WriteTo(out []byte) {
	out = out[:0]
	for _, b := range s {
		out = append(out, b.ReadOnlyData()...)
	}
}

func (s BufferSlice) Materialize() []byte {
	out := make([]byte, s.Len())
	s.WriteTo(out)
	return out
}

func (s BufferSlice) LazyMaterialize(pool BufferPool) *Buffer {
	if len(s) == 1 {
		return s[0].Ref()
	}
	buf := pool.Get(s.Len())
	s.WriteTo(buf)
	return NewBuffer(buf, pool.Put)
}
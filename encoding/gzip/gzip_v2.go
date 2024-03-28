package gzip

import (
	"bytes"
	"errors"
	"io"

	"google.golang.org/grpc/encoding"
)

var _ encoding.CompressorV2 = (*compressorV2)(nil)

type compressorV2 struct {
	c *compressor
}

func (c *compressorV2) Name() string {
	return Name
}

type yieldingWriter struct {
	yield func(encoding.Buffer, error) bool
}

var errSeqStopped = errors.New("")

func (s *yieldingWriter) Write(p []byte) (n int, err error) {
	buf := encoding.NewBuffer(len(p))
	copy(buf.Data(), p)
	if !s.yield(buf, nil) {
		return 0, errSeqStopped
	}
	return len(p), err
}

func (c *compressorV2) Compress(in encoding.BufferSeq) (out encoding.BufferSeq) {
	return func(yield func(encoding.Buffer, error) bool) {
		w, err := c.c.Compress(&yieldingWriter{yield})
		if err != nil {
			yield(nil, err)
			return
		}
		closed := false
		closeOnce := func() error {
		}

		in(func(in encoding.Buffer, err error) bool {
			if err != nil {
				return yield(nil, err)
			}
			defer in.Free()

			_, err = w.Write(in.Data())
			if err != nil {
				if !errors.Is(err, errSeqStopped) {
					return yield(nil, errors.Join(err, w.Close()))
				}
			}

			err = w.Close()
			if err != nil {
				c.freeBuffer(buf.Bytes())
				return yield(nil, err)
			}

			return yield(encoding.BufferFor(buf.Bytes(), c.freeBuffer), nil)
		})
	}
}

func (c *compressorV2) Decompress(in encoding.BufferSeq) (out encoding.BufferSeq) {
	return func(yield func(encoding.Buffer, error) bool) {
		in(func(in encoding.Buffer, err error) bool {
			if err != nil {
				return yield(nil, err)
			}
			defer in.Free()

			r, err := c.c.Decompress(bytes.NewReader(in.Data))
			if err != nil {
				return yield(nil, err)
			}

			buf := c.newBuffer()
			_, err = io.Copy(buf, r)
			if err != nil {
				c.freeBuffer(buf.Bytes())
				return yield(nil, err)
			}

			return yield(encoding.BufferFor(buf.Bytes(), c.freeBuffer), nil)
		})
	}
}

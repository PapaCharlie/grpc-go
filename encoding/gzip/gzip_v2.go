/*
 *
 * Copyright 2017 gRPC authors.
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

// Package gzip implements and registers the gzip compressor
// during the initialization.
//
// # Experimental
//
// Notice: This package is EXPERIMENTAL and may be changed or removed in a
// later release.
package gzip

import (
	"bytes"
	"errors"
	"io"
	"sync"

	"google.golang.org/grpc/encoding"
)

var _ encoding.CompressorV2 = (*compressorV2)(nil)

type compressorV2 struct {
	c          *compressor
	bufferPool sync.Pool
}

func (c *compressorV2) Name() string {
	return Name
}

func (c *compressorV2) freeBuffer(buf []byte) {
	c.bufferPool.Put(encoding.ClearBuffer(buf))
}

func (c *compressorV2) newBuffer() *bytes.Buffer {
	return bytes.NewBuffer(c.bufferPool.Get().([]byte))
}

func (c *compressorV2) Compress(in encoding.BufferSeq) (out encoding.BufferSeq) {
	return func(yield func(*encoding.Buffer, error) bool) {
		in(func(in *encoding.Buffer, err error) bool {
			if err != nil {
				return yield(nil, err)
			}
			defer in.Free()

			buf := c.newBuffer()
			w, err := c.c.Compress(buf)
			if err != nil {
				return yield(nil, err)
			}

			_, err = w.Write(in.Data)
			if err != nil {
				c.freeBuffer(buf.Bytes())
				return yield(nil, errors.Join(err, w.Close()))
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
	return func(yield func(*encoding.Buffer, error) bool) {
		in(func(in *encoding.Buffer, err error) bool {
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

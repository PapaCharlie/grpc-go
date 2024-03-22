/*
 *
 * Copyright 2018 gRPC authors.
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

// Package proto defines the protobuf codec. Importing this package will
// register the codec.
package proto

import (
	"fmt"
	"sync"

	"google.golang.org/grpc/encoding"
	"google.golang.org/protobuf/proto"
)

func init() {
	encoding.RegisterCodecV2(&codecV2{
		bufferPool: sync.Pool{New: func() any { return []byte(nil) }},
	})
}

// codec is an experimental.CodecV2 implementation with protobuf. It is the
// default codec for gRPC.
type codecV2 struct {
	bufferPool sync.Pool
}

func (c *codecV2) freeBuffer(buf []byte) {
	c.bufferPool.Put(encoding.ClearBuffer(buf))
}

func (c *codecV2) newBuffer() []byte {
	return c.bufferPool.Get().([]byte)
}

func (c *codecV2) Marshal(v any) (length int, out encoding.BufferSeq) {
	vv := messageV2Of(v)
	if vv == nil {
		return 0, func(yield func(*encoding.Buffer, error) bool) {
			yield(nil, fmt.Errorf("failed to marshal, message is %T, want proto.Message", v))
		}
	}
	buf := c.newBuffer()
	buf, err := proto.MarshalOptions{}.MarshalAppend(buf, vv)
	return len(buf), func(yield func(*encoding.Buffer, error) bool) {
		if err != nil {
			yield(nil, err)
		} else {
			yield(encoding.BufferFor(buf, c.freeBuffer), nil)
		}
	}
}

func (c *codecV2) GetBuffer(length int) *encoding.Buffer {
	buf := c.newBuffer()
	if cap(buf) < length {
		buf = make([]byte, length)
	}
	return encoding.BufferFor(buf, c.freeBuffer)
}

func (c *codecV2) Unmarshal(v any, length int, in encoding.BufferSeq) (err error) {
	vv := messageV2Of(v)
	if vv == nil {
		return fmt.Errorf("failed to unmarshal, message is %T, want proto.Message", v)
	}

	var out []byte

	in(func(buffer *encoding.Buffer, innerErr error) bool {
		if innerErr != nil {
			err = innerErr
			return false
		}
		defer buffer.Free()

		if out == nil {
			// Avoid copying anything if all the necessary data is in the initial buffer
			if len(buffer.Data) == length {
				out = buffer.Data
			} else {
				out = c.newBuffer()
			}
		}

		if len(out) < length {
			out = append(out, buffer.Data...)
		}

		if len(out) == length {
			err = proto.Unmarshal(buffer.Data, vv)
			return false
		} else {
			return true
		}
	})

	return err
}

func (c *codecV2) Name() string {
	return Name
}

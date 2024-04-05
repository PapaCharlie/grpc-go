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

	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/internal"
	"google.golang.org/protobuf/proto"
)

func init() {
	encoding.RegisterCodecV2(&codecV2{pool: internal.NewSharedBufferPool()})
}

// codec is an experimental.CodecV2 implementation with protobuf. It is the
// default codec for gRPC.
type codecV2 struct {
	pool internal.SharedBufferPool
}

var _ encoding.CodecV2 = (*codecV2)(nil)
var _ encoding.BufferProvider = (*codecV2)(nil)

func (c *codecV2) Marshal(v any) ([][]byte, error) {
	vv := messageV2Of(v)
	if vv == nil {
		return nil, fmt.Errorf("proto: failed to marshal, message is %T, want proto.Message", v)
	}

	buf := c.GetBuffer(proto.Size(vv))
	_, err := proto.MarshalOptions{}.MarshalAppend(buf[:0], vv)
	if err != nil {
		c.ReturnBuffer(buf)
		return nil, err
	} else {
		return [][]byte{buf}, nil
	}
}

func (c *codecV2) Unmarshal(v any, data [][]byte) (err error) {
	vv := messageV2Of(v)
	if vv == nil {
		return fmt.Errorf("failed to unmarshal, message is %T, want proto.Message", v)
	}

	buf := encoding.ConcatBuffersSlice(data, c)
	defer c.ReturnBuffer(buf)

	return proto.Unmarshal(buf, vv)
}

func (c *codecV2) Name() string {
	return Name
}

func (c *codecV2) GetBuffer(length int) []byte {
	return c.pool.Get(length)
}

func (c *codecV2) ReturnBuffer(buf []byte) {
	c.pool.Put(&buf)
}

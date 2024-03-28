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
	"google.golang.org/protobuf/proto"
)

func init() {
	encoding.RegisterCodecV2(new(codecV2))
}

// codec is an experimental.CodecV2 implementation with protobuf. It is the
// default codec for gRPC.
type codecV2 struct {
}

func (c *codecV2) Marshal(v any) *encoding.BufferSeq {
	vv := messageV2Of(v)
	if vv == nil {
		return encoding.ErrBufferSeq(fmt.Errorf("proto: failed to marshal, message is %T, want proto.Message", v))
	}

	buf := encoding.NewBuffer(proto.Size(vv))
	data, err := proto.MarshalOptions{}.MarshalAppend(buf.Data()[:0], vv)
	if err != nil {
		buf.Free()
		buf = nil
	} else {
		buf.SetData(data)
	}

	return encoding.OneElementSeq(len(data), buf, err)
}

func (c *codecV2) GetBuffer(length int) encoding.Buffer {
	return encoding.NewBuffer(length)
}

func (c *codecV2) Unmarshal(v any, data *encoding.BufferSeq) (err error) {
	vv := messageV2Of(v)
	if vv == nil {
		return fmt.Errorf("failed to unmarshal, message is %T, want proto.Message", v)
	}

	buf, err := encoding.FullRead(data, encoding.NewBuffer)
	if err != nil {
		return err
	}
	defer buf.Free()

	return proto.Unmarshal(buf.Data(), vv)
}

func (c *codecV2) Name() string {
	return Name
}

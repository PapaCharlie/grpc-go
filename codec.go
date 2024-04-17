/*
 *
 * Copyright 2014 gRPC authors.
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

package grpc

import (
	"google.golang.org/grpc/encoding"
	_ "google.golang.org/grpc/encoding/proto" // to register the Codec for "proto"
	"google.golang.org/grpc/mem"
)

// baseCodec captures the new encoding.CodecV2 interface without the Name
// function, allowing it to be implemented by older Codec and encoding.Codec
// implementations. The omitted Name function is only needed for the register in
// the encoding package and is not part of the core functionality.
type baseCodec interface {
	Marshal(v any) (mem.BufferSlice, error)
	Unmarshal(data mem.BufferSlice, v any) error
}

type namedBaseCodec interface {
	baseCodec
	Name() string
}

func getCodec(name string) namedBaseCodec {
	codecV2 := encoding.GetCodecV2(name)
	if codecV2 != nil {
		return codecV2
	}

	codecV1 := encoding.GetCodec(name)
	if codecV1 != nil {
		return newCodecV1Bridge(codecV1)
	}

	return nil
}

func newCodecV0Bridge(c Codec) baseCodec {
	return codecV0Bridge{c}
}

func newCodecV1Bridge(c encoding.Codec) namedBaseCodec {
	return codecV1Bridge{
		codecV0Bridge: codecV0Bridge{c},
		name:          c.Name(),
	}
}

var _ baseCodec = codecV0Bridge{}

type codecV0Bridge struct {
	codec interface {
		Marshal(v any) ([]byte, error)
		Unmarshal(data []byte, v any) error
	}
}

func (c codecV0Bridge) Marshal(v any) (mem.BufferSlice, error) {
	data, err := c.codec.Marshal(v)
	if err != nil {
		return nil, err
	} else {
		return mem.BufferSlice{mem.NewBuffer(data, nil)}, nil
	}
}

func (c codecV0Bridge) Unmarshal(data mem.BufferSlice, v any) (err error) {
	return c.codec.Unmarshal(data.Materialize(), v)
}

var _ namedBaseCodec = codecV1Bridge{}

type codecV1Bridge struct {
	codecV0Bridge
	name string
}

func (c codecV1Bridge) Name() string {
	return c.name
}

// Codec defines the interface gRPC uses to encode and decode messages.
// Note that implementations of this interface must be thread safe;
// a Codec's methods can be called from concurrent goroutines.
//
// Deprecated: use encoding.Codec instead.
type Codec interface {
	// Marshal returns the wire format of v.
	Marshal(v any) ([]byte, error)
	// Unmarshal parses the wire format into v.
	Unmarshal(data []byte, v any) error
	// String returns the name of the Codec implementation.  This is unused by
	// gRPC.
	String() string
}

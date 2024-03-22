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

package encoding

import (
	"strings"

	"google.golang.org/grpc/internal/grpcutil"
)

type Buffer struct {
	Data []byte
	free func([]byte)
}

func BufferFor(buf []byte, free func([]byte)) *Buffer {
	return &Buffer{Data: buf, free: free}
}

func ClearBuffer(buf []byte) []byte {
	// TODO: replace with clear when go1.21 is supported
	// clear(buf)
	for i := range buf {
		buf[i] = 0
	}
	buf = buf[:0]
	return buf
}

func (b *Buffer) Free() {
	if b.free != nil {
		b.free(b.Data)
	}
}

// BufferSeq is the equivalent of iter.Seq[*Buffer, error], but cannot be added
// by directly referencing the new iter package since it is not yet supported in
// all versions of go supported by grpc-go.
type BufferSeq = func(yield func(*Buffer, error) bool)

// CompressorV2 is used for compressing and decompressing when sending or
// receiving messages.
type CompressorV2 interface {
	// Compress returns a BufferSeq containing the compressed data of the input
	// BufferSeq. An error can be returned at any point in the sequence, terminating
	// the execution.
	Compress(in BufferSeq) (out BufferSeq)
	// Decompress reads data from the input BufferSeq, decompresses it, and provides
	// the uncompressed data via the returned BufferSeq. An error can be returned at
	// any point in the sequence, terminating the execution.
	Decompress(in BufferSeq) (out BufferSeq)
	// Name is the name of the compression codec and is used to set the content
	// coding header.  The result must be static; the result cannot change
	// between calls.
	Name() string
}

var registeredCompressorV2 = make(map[string]CompressorV2)

// RegisterCompressorV2 registers the compressor with gRPC by its name. It can be
// activated when sending an RPC via grpc.UseCompressor(). It will be
// automatically accessed when receiving a message based on the content coding
// header. Servers also use it to send a response with the same encoding as the
// request.
//
// NOTE: this function must only be called during initialization time (i.e. in
// an init() function), and is not thread-safe.  If multiple Compressors are
// registered with the same name, the one registered last will take effect.
func RegisterCompressorV2(c CompressorV2) {
	registeredCompressorV2[c.Name()] = c
	if !grpcutil.IsCompressorNameRegistered(c.Name()) {
		grpcutil.RegisteredCompressorNames = append(grpcutil.RegisteredCompressorNames, c.Name())
	}
}

// GetCompressorV2 returns the CompressorV2 for the given compressor name, or nil
// if no compressor by that name was registered.
func GetCompressorV2(name string) CompressorV2 {
	return registeredCompressorV2[name]
}

// CodecV2 defines the interface gRPC uses to encode and decode messages. Note
// that implementations of this interface must be thread safe; a CodecV2's
// methods can be called from concurrent goroutines.
type CodecV2 interface {
	Marshal(v any) (length int, out BufferSeq)
	GetBuffer(length int) *Buffer
	Unmarshal(v any, length int, in BufferSeq) error
	// Name returns the name of the Codec implementation. The returned string
	// will be used as part of content type in transmission.  The result must be
	// static; the result cannot change between calls.
	Name() string
}

var registeredCodecsV2 = make(map[string]CodecV2)

// RegisterCodecV2 registers the provided CodecV2 for use with all gRPC clients and
// servers.
//
// The CodecV2 will be stored and looked up by result of its Name() method, which
// should match the content-subtype of the encoding handled by the CodecV2.  This
// is case-insensitive, and is stored and looked up as lowercase.  If the
// result of calling Name() is an empty string, RegisterCodecV2 will panic. See
// Content-Type on
// https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md#requests for
// more details.
//
// NOTE: this function must only be called during initialization time (i.e. in
// an init() function), and is not thread-safe.  If multiple Codecs are
// registered with the same name, the one registered last will take effect.
func RegisterCodecV2(codec CodecV2) {
	if codec == nil {
		panic("cannot register a nil CodecV2")
	}
	if codec.Name() == "" {
		panic("cannot register CodecV2 with empty string result for Name()")
	}
	contentSubtype := strings.ToLower(codec.Name())
	registeredCodecsV2[contentSubtype] = codec
}

// GetCodecV2 gets a registered CodecV2 by content-subtype, or nil if no CodecV2 is
// registered for the content-subtype.
//
// The content-subtype is expected to be lowercase.
func GetCodecV2(contentSubtype string) CodecV2 {
	return registeredCodecsV2[contentSubtype]
}

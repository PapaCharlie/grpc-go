package bridge

import (
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/internal"
)

type BaseCodecV2 interface {
	Marshal(v any) ([][]byte, error)
	Unmarshal(data [][]byte, v any) error
}

type CodecV1Bridge struct {
	Codec interface {
		Marshal(v any) ([]byte, error)
		Unmarshal(data []byte, v any) error
	}
}

func (c CodecV1Bridge) GetBuffer(size int) []byte {
	return pool.GetBuffer(size)
}

func (c CodecV1Bridge) ReturnBuffer(buf []byte) {
	pool.ReturnBuffer(buf)
}

func (c CodecV1Bridge) Marshal(v any) ([][]byte, error) {
	data, err := c.Codec.Marshal(v)
	if err != nil {
		return nil, err
	} else {
		return [][]byte{data}, nil
	}
}

var pool = internal.NewSharedBufferPool()

func (c CodecV1Bridge) Unmarshal(data [][]byte, v any) (err error) {
	buf := pool.GetBuffer(encoding.BufferSliceSize(data))
	defer pool.ReturnBuffer(buf)
	encoding.WriteBufferSlice(data, buf)
	return c.Codec.Unmarshal(buf, v)
}

func GetCodec(name string) BaseCodecV2 {
	var codec BaseCodecV2
	codec = encoding.GetCodecV2(name)
	if codec == nil {
		codecV1 := encoding.GetCodec(name)
		if codecV1 != nil {
			codec = CodecV1Bridge{Codec: codecV1}
		}
	}
	return codec
}

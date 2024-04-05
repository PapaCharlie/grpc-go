package internalencoding

import (
	"google.golang.org/grpc/encoding"
)

type BaseCodecV2 interface {
	Marshal(v any) ([][]byte, error)
	Unmarshal(v any, data [][]byte) error
}

type CodecV1Bridge struct {
	Codec interface {
		Marshal(v any) ([]byte, error)
		Unmarshal(data []byte, v any) error
	}
}

func (c CodecV1Bridge) Marshal(v any) ([][]byte, error) {
	data, err := c.Codec.Marshal(v)
	if err != nil {
		return nil, err
	} else {
		return [][]byte{data}, nil
	}
}

func (c CodecV1Bridge) Unmarshal(v any, data [][]byte) (err error) {
	return c.Codec.Unmarshal(encoding.ConcatBuffersSlice(data, nil), v)
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

package internalencoding

import (
	"google.golang.org/grpc/encoding"
)

type BaseCodecV2 interface {
	Marshal(v any) (encoding.BufferSeq, error)
	GetBuffer(length int) encoding.Buffer
	Unmarshal(v any, data encoding.BufferSeq) error
}

type CodecV1Bridge struct {
	Codec interface {
		Marshal(v any) ([]byte, error)
		Unmarshal(data []byte, v any) error
	}
}

func (c CodecV1Bridge) Marshal(v any) (encoding.BufferSeq, error) {
	data, err := c.Codec.Marshal(v)
	if err != nil {
		return nil, err
	} else {
		return encoding.BufferSeq{encoding.SimpleBuffer(data)}, nil
	}
}

func (c CodecV1Bridge) GetBuffer(length int) encoding.Buffer {
	return encoding.NewBuffer(length)
}

func (c CodecV1Bridge) Unmarshal(v any, data encoding.BufferSeq) (err error) {
	buf := data.Concat(encoding.NewBuffer)
	defer buf.Free()
	return c.Codec.Unmarshal(buf.Data(), v)
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

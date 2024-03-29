package internalencoding

import (
	"google.golang.org/grpc/encoding"
)

type BaseCodecV2 interface {
	Marshal(v any) (int, encoding.BufferSeq)
	GetBuffer(length int) encoding.Buffer
	Unmarshal(v any, length int, data encoding.BufferSeq) error
}

type CodecV1Bridge struct {
	Codec interface {
		Marshal(v any) ([]byte, error)
		Unmarshal(data []byte, v any) error
	}
}

type noopBuffer struct {
	data []byte
}

func (n *noopBuffer) Data() []byte {
	return n.data
}

func (n *noopBuffer) SetData(data []byte) {
	n.data = data
}

func (n *noopBuffer) Free() {}

func (c CodecV1Bridge) Marshal(v any) (int, encoding.BufferSeq) {
	data, err := c.Codec.Marshal(v)
	var buf encoding.Buffer
	if err == nil {
		buf = &noopBuffer{data}
	}
	return len(data), func(yield func(encoding.Buffer, error) bool) {
		yield(buf, err)
	}
}

func (c CodecV1Bridge) GetBuffer(length int) encoding.Buffer {
	return encoding.NewBuffer(length)
}

func (c CodecV1Bridge) Unmarshal(v any, length int, data encoding.BufferSeq) (err error) {
	buf, err := encoding.FullRead(length, data, encoding.NewBuffer)
	if err != nil {
		return err
	}
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

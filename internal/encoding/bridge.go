package internalencoding

import (
	"google.golang.org/grpc/encoding"
)

type BaseCodec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

type BaseCodecV2 interface {
	Marshal(v any) *encoding.BufferSeq
	GetBuffer(length int) encoding.Buffer
	Unmarshal(v any, data *encoding.BufferSeq) error
}

type CodecV1Bridge struct {
	BaseCodec
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

func (c CodecV1Bridge) Marshal(v any) *encoding.BufferSeq {
	data, err := c.BaseCodec.Marshal(v)
	var buf encoding.Buffer
	if err == nil {
		buf = &noopBuffer{data}
	}
	return &encoding.BufferSeq{
		Len: len(data),
		Seq: func(yield func(encoding.Buffer, error) bool) {
			yield(buf, err)
		},
	}
}

func (c CodecV1Bridge) GetBuffer(length int) encoding.Buffer {
	return encoding.NewBuffer(length)
}

func (c CodecV1Bridge) Unmarshal(v any, data *encoding.BufferSeq) (err error) {
	buf, err := encoding.FullRead(data, encoding.NewBuffer)
	if err != nil {
		return err
	}
	defer buf.Free()
	return c.BaseCodec.Unmarshal(buf.Data(), v)
}

func GetCodec(name string) BaseCodecV2 {
	var codec BaseCodecV2
	codec = encoding.GetCodecV2(name)
	if codec == nil {
		codecV1 := encoding.GetCodec(name)
		if codecV1 != nil {
			codec = CodecV1Bridge{BaseCodec: codecV1}
		}
	}
	return codec
}

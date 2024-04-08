package bridge

import (
	"io"

	"google.golang.org/grpc/encoding"
)

type BaseDecompressorV2 interface {
	Decompress(in [][]byte, provider encoding.BufferProvider) (out [][]byte, err error)
	Name() string
}

type decompressorV0 interface {
	Do(r io.Reader) ([]byte, error)
	Type() string
}

func DecompressorV0Bridge(v0 decompressorV0) BaseDecompressorV2 {
	return decompressorV0Bridge{v0}
}

type decompressorV0Bridge struct {
	decompressorV0
}

func (d decompressorV0Bridge) Name() string {
	return d.decompressorV0.Type()
}

func (d decompressorV0Bridge) Decompress(in [][]byte, _ encoding.BufferProvider) ([][]byte, error) {
	out, err := d.decompressorV0.Do(encoding.NewBufferSliceReader(in))
	return [][]byte{out}, err
}

type decompressorV1 interface {
	Decompress(r io.Reader) (io.Reader, error)
	Name() string
}

func DecompressorV1Bridge(v1 decompressorV1) BaseDecompressorV2 {
	return decompressorV1Bridge{decompressorV1: v1}
}

type decompressorV1Bridge struct {
	decompressorV1
}

func (d decompressorV1Bridge) Decompress(in [][]byte, provider encoding.BufferProvider) (out [][]byte, err error) {
	r, err := d.decompressorV1.Decompress(encoding.NewBufferSliceReader(in))
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(encoding.NewBufferSliceWriter(&out, provider), r)
	if err != nil {
		encoding.ReturnAllBuffers(out, provider)
		return nil, err
	}

	return out, nil
}

func GetDecompressor(name string) BaseDecompressorV2 {
	var dc BaseDecompressorV2
	dc = encoding.GetCompressorV2(name)
	if dc == nil {
		dcV1 := encoding.GetCompressor(name)
		if dcV1 != nil {
			dc = DecompressorV1Bridge(dcV1)
		}
	}
	return dc
}

package bridge

import (
	"bytes"
	"io"

	"google.golang.org/grpc/encoding"
)

type BaseCompressorV2 interface {
	Compress(in [][]byte) (out [][]byte, err error)
	Name() string
}

type compressorV0 interface {
	Do(w io.Writer, p []byte) error
	Type() string
}

func CompressorV0Bridge(v0 compressorV0) BaseCompressorV2 {
	return compressorV0Bridge{v0}
}

type compressorV0Bridge struct {
	compressorV0
}

func (c compressorV0Bridge) GetBuffer(size int) []byte {
	return pool.Get(size)
}

func (c compressorV0Bridge) ReturnBuffer(buf []byte) {
	pool.ReturnBuffer(buf)
}

func (c compressorV0Bridge) Name() string {
	return c.Type()
}

func (c compressorV0Bridge) Compress(in [][]byte) ([][]byte, error) {
	buf := pool.GetBuffer(encoding.BufferSliceSize(in))
	defer pool.ReturnBuffer(buf)
	encoding.WriteBufferSlice(in, buf)

	out := bytes.NewBuffer(nil)
	err := c.Do(out, buf)
	if err != nil {
		return nil, err
	}
	return [][]byte{out.Bytes()}, nil
}

type compressorV1 interface {
	Compress(w io.Writer) (io.WriteCloser, error)
	Name() string
}

func CompressorV1Bridge(v1 compressorV1) BaseCompressorV2 {
	return compressorV1Bridge{v1}
}

type compressorV1Bridge struct {
	compressorV1
}

func (c compressorV1Bridge) GetBuffer(size int) []byte {
	return pool.Get(size)
}

func (c compressorV1Bridge) ReturnBuffer(buf []byte) {
	pool.ReturnBuffer(buf)
}

func (c compressorV1Bridge) Compress(in [][]byte) (out [][]byte, err error) {
	w, err := c.compressorV1.Compress(encoding.NewBufferSliceWriter(&out, c))
	if err != nil {
		return nil, err
	}

	returnAll := func() {
		for _, b := range out {
			c.ReturnBuffer(b)
		}
	}

	for _, b := range in {
		_, err = w.Write(b)
		if err != nil {
			returnAll()
			return nil, err
		}
	}

	err = w.Close()
	if err != nil {
		returnAll()
		return nil, err
	}

	return out, nil
}

func GetCompressor(name string) BaseCompressorV2 {
	var comp BaseCompressorV2
	comp = encoding.GetCompressorV2(name)
	if comp == nil {
		compV1 := encoding.GetCompressor(name)
		if compV1 != nil {
			comp = CompressorV1Bridge(compV1)
		}
	}
	return comp
}

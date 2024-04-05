package internalencoding

import (
	"bytes"
	"io"

	"google.golang.org/grpc/encoding"
)

type BaseCompressorV2 interface {
	Compress(in [][]byte) (out [][]byte, err error)
}

type CompressorV0Bridge struct {
	Compressor interface {
		Do(w io.Writer, p []byte) error
	}
}

func (c CompressorV0Bridge) Compress(in [][]byte) (out [][]byte, err error) {
	buf := new(bytes.Buffer)
	err = c.Compressor.Do(buf, encoding.ConcatBufferSlice(in, nil))
	if err != nil {
		return nil, err
	}
	return [][]byte{buf.Bytes()}, nil
}

type CompressorV1Bridge struct {
	Compressor interface {
		Compress(w io.Writer) (io.WriteCloser, error)
	}
}

func (c CompressorV1Bridge) Compress(in [][]byte) (out [][]byte, err error) {
	w, err := c.Compressor.Compress(encoding.BufferSliceWriter(&out, nil))
	if err != nil {
		return nil, err
	}

	for _, b := range in {
		_, err = w.Write(b)
		if err != nil {
			return nil, err
		}
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	return out, nil
}

type BaseDecompressorV2 interface {
	Decompress(in [][]byte, provider encoding.BufferProvider) (out [][]byte, err error)
}

type DecompressorV0Bridge struct {
	Decompressor interface {
		Do(r io.Reader) ([]byte, error)
	}
}

func (d DecompressorV0Bridge) Decompress(in [][]byte, provider encoding.BufferProvider) (out [][]byte, err error) {
	//TODO implement me
	panic("implement me")
}

type DecompressorV1Bridge struct {
	Decompressor interface {
		Decompress(r io.Reader) (io.Reader, error)
	}
}

func (d DecompressorV1Bridge) Decompress(in [][]byte, provider encoding.BufferProvider) (out [][]byte, err error) {
	//TODO implement me
	panic("implement me")
}

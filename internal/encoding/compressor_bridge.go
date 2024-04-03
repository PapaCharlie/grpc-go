package internalencoding

import (
	"bytes"
	"io"

	"google.golang.org/grpc/encoding"
)

type BaseCompressorV2 interface {
	Compress(in encoding.BufferSeq) (out encoding.BufferSeq, err error)
}

type CompressorV0Bridge struct {
	Compressor interface {
		Do(w io.Writer, p []byte) error
	}
}

func (c CompressorV0Bridge) Compress(in encoding.BufferSeq) (out encoding.BufferSeq, err error) {
	data := in.Concat(encoding.NewBuffer)
	defer data.Free()

	buf := new(bytes.Buffer)
	err = c.Compressor.Do(buf, data.Data())
	if err != nil {
		return nil, err
	}
	return encoding.BufferSeq{encoding.SimpleBuffer(buf.Bytes())}, nil
}

type CompressorV1Bridge struct {
	Compressor interface {
		Compress(w io.Writer) (io.WriteCloser, error)
	}
}

type seqWriter encoding.BufferSeq

func (s *seqWriter) Write(data []byte) (int, error) {
	buf := encoding.NewBuffer(len(data))
	copy(buf.Data(), data)
	*s = append(*s, buf)
	return len(data), nil
}

func (c CompressorV1Bridge) Compress(in encoding.BufferSeq) (out encoding.BufferSeq, err error) {
	defer func() {
		if err != nil {
			out.Free()
		}
	}()

	w, err := c.Compressor.Compress((*seqWriter)(&out))
	if err != nil {
		return nil, err
	}

	for _, b := range in {
		_, err = w.Write(b.Data())
		b.Free()
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
	GetBuffer(length int) encoding.Buffer
	Decompress(in encoding.BufferSeq) (out encoding.BufferSeq, err error)
}

type DecompressorV0Bridge struct {
	Decompressor interface {
		Do(r io.Reader) ([]byte, error)
	}
}

type DecompressorV1Bridge struct {
	Decompressor interface {
		Decompress(r io.Reader) (io.Reader, error)
	}
}

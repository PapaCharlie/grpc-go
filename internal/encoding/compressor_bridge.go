package internalencoding

import (
	"bytes"
	"io"

	"google.golang.org/grpc/encoding"
)

type BaseCompressorV2 interface {
	Compress(in *MaterializedBufferSeq) (out *MaterializedBufferSeq, err error)
}

type CompressorV0Bridge struct {
	Compressor interface {
		Do(w io.Writer, p []byte) error
	}
}

func (c CompressorV0Bridge) Compress(in *MaterializedBufferSeq) (out *MaterializedBufferSeq, err error) {
	data := encoding.NewBuffer(in.Len)
	in.Read(data.Data())
	defer data.Free()

	buf := new(bytes.Buffer)
	err = c.Compressor.Do(buf, data.Data())
	if err != nil {
		return nil, err
	}
	return &MaterializedBufferSeq{
		Len:  buf.Len(),
		Data: []encoding.Buffer{encoding.SimpleBuffer(buf.Bytes())},
	}, nil
}

type CompressorV1Bridge struct {
	Compressor interface {
		Compress(w io.Writer) (io.WriteCloser, error)
	}
}

type seqWriter struct {
	seq *MaterializedBufferSeq
}

func (s *seqWriter) Write(data []byte) (n int, err error) {
	buf := encoding.NewBuffer(len(data))
	s.seq.Data = append(s.seq.Data, buf)
	n = copy(buf.Data(), data)
	s.seq.Len += n
	return n, nil
}

func (c CompressorV1Bridge) Compress(in *MaterializedBufferSeq) (out *MaterializedBufferSeq, err error) {
	out = new(MaterializedBufferSeq)

	defer func() {
		if err != nil {
			out.Free()
		}
	}()

	w, err := c.Compressor.Compress(&seqWriter{seq: out})
	if err != nil {
		return nil, err
	}

	for _, b := range in.Data {
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
	Decompress(in *MaterializedBufferSeq) (out *MaterializedBufferSeq, err error)
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

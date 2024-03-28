package internalencoding

import (
	"fmt"

	"google.golang.org/grpc/encoding"
)

func MaterializeBufferSeq(data *encoding.BufferSeq) (m *MaterializedBufferSeq, err error) {
	m = &MaterializedBufferSeq{}
	data.Seq(func(buf encoding.Buffer, innerErr error) bool {
		if innerErr != nil {
			innerErr = err
			return false
		}
		m.Data = append(m.Data, buf)
		m.Len += len(buf.Data())
		return true
	})

	if err != nil {
		return nil, err
	}

	if m.Len != data.Len {
		return nil, fmt.Errorf("grpc: too many bytes received from BufferSeq, expected %d got %d", data.Len, m.Len)
	}

	return m, nil
}

type MaterializedBufferSeq struct {
	Len  int
	Data []encoding.Buffer
}

func (m *MaterializedBufferSeq) Read(buf []byte) {
	for len(buf) != 0 && m.Len != 0 {
		data := m.Data[0]
		copied := copy(buf, data.Data())
		m.Len -= copied

		buf = buf[copied:]

		if copied == len(data.Data()) {
			data.Free()
			m.Data = m.Data[1:]
		} else {
			data.SetData(data.Data()[copied:])
		}
	}
}

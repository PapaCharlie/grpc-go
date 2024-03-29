package internalencoding

import (
	"fmt"

	"google.golang.org/grpc/encoding"
)

func MaterializeBufferSeq(length int, data encoding.BufferSeq) (m *MaterializedBufferSeq, err error) {
	m = &MaterializedBufferSeq{}
	data(func(buf encoding.Buffer, innerErr error) bool {
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

	if m.Len != length {
		return nil, fmt.Errorf("grpc: unexpected byte count from BufferSeq, expected %d got %d", length, m.Len)
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

func (m *MaterializedBufferSeq) ReadFull() []byte {
	buf := make([]byte, m.Len)
	m.Read(buf)
	return buf
}

func (m *MaterializedBufferSeq) Free() {
	for _, b := range m.Data {
		b.Free()
	}
}

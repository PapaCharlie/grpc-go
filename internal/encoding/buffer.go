package internalencoding

type Buffer struct {
	Data []byte

	refCount int
	free     func([]byte)
}

func (b *Buffer) Free() {
	b.free(b.Data)
}

func NewBuffer(data []byte, free func([]byte)) *Buffer {
	return &Buffer{
		Data: data,
		free: free,
	}
}

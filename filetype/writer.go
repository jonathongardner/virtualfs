package filetype

// mime type wrapper that allows to copy

import (
	"github.com/gabriel-vasile/mimetype"
	// log "github.com/sirupsen/logrus"
)

type FiletypeWriter struct {
	position int
	data     []byte
}

func NewFiletypeWriter() *FiletypeWriter {
	return &FiletypeWriter{position: 0, data: make([]byte, 0)}
}

func (mw *FiletypeWriter) Write(p []byte) (n int, err error) {
	len := len(p)
	toCopy := 0

	if mw.position < maxBytesFileDetect {
		toCopy = maxBytesFileDetect - mw.position

		if toCopy > len {
			toCopy = len
		}

		mw.position += toCopy
		mw.data = append(mw.data, p[:toCopy]...)
	}

	return len, nil
}

func (mw *FiletypeWriter) String() Filetype {
	return newFiletype(mimetype.Detect(mw.data))
}

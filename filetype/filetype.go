package filetype

import (
	"io"

	"github.com/gabriel-vasile/mimetype"
	// log "github.com/sirupsen/logrus"
)

// https://github.com/gabriel-vasile/mimetype/blob/master/mimetype.go#L17
const maxBytesFileDetect int = 65536 // 2^16

func init() {
	mimetype.SetLimit(uint32(maxBytesFileDetect))
}

type Filetype struct {
	Extension string `json:"extension"`
	Mimetype  string `json:"mimetype"`
}

var Dir = Filetype{Extension: "dir", Mimetype: "directory/directory"}
var Symlink = Filetype{Extension: "symlink", Mimetype: "symlink/symlink"}

func newFiletype(mtype *mimetype.MIME) Filetype {
	return Filetype{Extension: mtype.Extension(), Mimetype: mtype.String()}
}

func NewFiletypeFromReader(reader io.Reader) (Filetype, error) {
	data := make([]byte, maxBytesFileDetect)
	w, err := reader.Read(data)
	if err != nil {
		return Filetype{}, err
	}
	if w < maxBytesFileDetect {
		data = data[:w]
	}
	return newFiletype(mimetype.Detect(data)), nil
}

func FiletypeFromJson(v map[string]any) Filetype {
	return Filetype{Extension: v["extension"].(string), Mimetype: v["mimetype"].(string)}
}

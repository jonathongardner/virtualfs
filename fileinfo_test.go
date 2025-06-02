package virtualfs

import (
	"os"
	"testing"
)

func TestFileInfoImplementsOSFileInfo(t *testing.T) {
	n := &Fs{name: "foo"}
	var v interface{} = n
	_, ok := v.(os.FileInfo)
	assert(t, ok, "expected FileInfo to implement os.FileInfo")
}

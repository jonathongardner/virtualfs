package virtualfs

import (
	"os"
	"testing"
)

func TestFileInfoImplementsOSFileInfo(t *testing.T) {
	myT := NewMyT("Test FileInfo implements os.FileInfo", t)

	n := &FileInfo{name: "foo"}
	var v interface{} = n
	_, ok := v.(os.FileInfo)
	myT.Assert(ok, "expected FileInfo to implement os.FileInfo")
	// cool := func(f os.FileInfo) {
	// 	// do nothing
	// }
	// cool(n)
}

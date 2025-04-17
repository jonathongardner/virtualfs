package virtualfs

import "fmt"

var ErrClosed = fmt.Errorf("virtual file system is closed")
var ErrChild = fmt.Errorf("virtual file system is a child of another fs")
var ErrReadOnly = fmt.Errorf("virtual file system is read only")
var ErrCantWriteNewFile = fmt.Errorf("cant write new file")

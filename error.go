package virtualfs

import "fmt"

var ErrClosed = fmt.Errorf("virtual file system is closed")
var ErrChild = fmt.Errorf("virtual file system is a child of another fs")

// var root = &Entry{type: filetype.Directory}
var ErrDontWalk = fmt.Errorf("dont walk entries children")
var ErrNotFound = fmt.Errorf("file not found") // https://smyrman.medium.com/writing-constant-errors-with-go-1-13-10c4191617
var ErrOutsideFilesystem = fmt.Errorf("path is outside of filesystem")
var ErrInFilesystem = fmt.Errorf("filesystem errors")

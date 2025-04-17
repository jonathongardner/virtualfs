package virtualfs

import (
	"fmt"
	"os"
	// log "github.com/sirupsen/logrus"
)

type Fs struct {
	root   *FileInfo
	parent *Fs
}

// var root = &Entry{type: filetype.Directory}
var ErrDontWalk = fmt.Errorf("dont walk entries children")
var ErrNotFound = fmt.Errorf("file not found") // https://smyrman.medium.com/writing-constant-errors-with-go-1-13-10c4191617
var ErrOutsideFilesystem = fmt.Errorf("path is outside of filesystem")
var ErrInFilesystem = fmt.Errorf("filesystem errors")

// NewFsFromDb loads a virtual file system from a directory DB
func NewFsFromDb(storageFile string, readonly bool) (*Fs, error) {
	options := os.O_RDWR
	if readonly {
		options = os.O_RDONLY
	}
	// Open as read only becasue we dont want to allow changing
	f, err := os.OpenFile(storageFile, options, 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to create output file: %v", err)
	}
	if readonly {
		defer f.Close()
	}

	toReturn := &Fs{}
	return toReturn, toReturn.load(f, readonly)
}

// NewFs creates a new virtual file system from a file or stdin
func NewFsFromFileInfo(storageFile string, fileInfo os.FileInfo, overwrite bool) (*Fs, error) {
	// Dont append b/c when saving need to rewrite to start of file
	options := os.O_RDWR | os.O_CREATE
	if overwrite {
		options |= os.O_TRUNC
	}
	f, err := os.OpenFile(storageFile, options, 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to create output file: %v", err)
	}

	fs := &Fs{root: newFileInfo(newReferenceDB(f), fileInfo.Name(), fileInfo.Mode(), fileInfo.ModTime())}
	if fileInfo.IsDir() {
		fs.root.setToDir()
	}

	return fs, nil
}

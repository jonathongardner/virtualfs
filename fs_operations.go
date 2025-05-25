package virtualfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonathongardner/fifo/filetype"
	// log "github.com/sirupsen/logrus"
)

// Stat returns an os.FileInfo like object (Fs) for the path
func (v *Fs) Stat(path string) (*Fs, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	toWalk, _, err := v.fsFrom(path, -1)
	if err != nil {
		return nil, err
	}
	return toWalk, nil
}

// StatAt returns an os.FileInfo like object for the path and index (if there are multiple, like for compression)
func (v *Fs) StatAt(path string, at int) (*Fs, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	toWalk, _, err := v.fsFrom(path, at)
	if err != nil {
		return nil, err
	}
	return toWalk, nil
}

// MkdirP creates a directory and all the parents if they do not exist
func (v *Fs) MkdirP(path string, perm os.FileMode, modTime time.Time) (*Fs, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	paths, err := split(path)
	if err != nil {
		return nil, err
	}
	newRoot, err := v.mkdirP(paths, perm, modTime)
	return newRoot, err
}

// CreateFile creates a new file in the storage directory
// when the file is closed if it matches another file
// (based on sha256) then it will be linked to that file
func (n *Fs) CreateFile() (*myFile, error) {
	return createMyWriterCloser(n, n.FilePath())
}

// OpenFile opens the Fs base file for reading
// returns an error if the file is a directory or a symlink
func (n *Fs) OpenFile() (*os.File, error) {
	if n.ref.typ == filetype.Symlink {
		return nil, fmt.Errorf("cannot open a symlink")
	}
	if n.ref.typ == filetype.Dir {
		return nil, fmt.Errorf("cannot open a directory")
	}

	return n.ref.open(n.db.storageDir)
}

// FilePath returns the path to the file in the storage directory
func (n *Fs) FilePath() string {
	return n.ref.storagePath(n.db.storageDir)
}

// Open returns an os.File for the path, if no path is given, it will return the root
func (v *Fs) Open(path string) (*os.File, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	toWalk, _, err := v.fsFrom(path, -1)
	if err != nil {
		return nil, err
	}
	return toWalk.OpenFile()
}

// CreateWithoutPath creates a child file
func (v *Fs) CreateWithoutPath(perm os.FileMode, modTime time.Time) (*Fs, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	newFileInfo, err := v.touch([]string{}, perm, modTime)
	return newFileInfo, err
}

// Create creates a file to the path
func (v *Fs) Create(path string, perm os.FileMode, modTime time.Time) (*Fs, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	paths, err := split(path)
	if err != nil {
		return nil, err
	}
	if len(path) == 0 {
		return nil, ErrOutsideFilesystem
	}

	newFileInfo, err := v.touch(paths, perm, modTime)
	return newFileInfo, err
}

// Symlink creates a symlnk at the path
// source is the path to the file to link to
func (v *Fs) Symlink(source, newname string, perm os.FileMode, modTime time.Time) (*Fs, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	paths, err := split(newname)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, ErrOutsideFilesystem
	}

	newRoot, err := v.symlink(source, paths, perm, modTime)
	return newRoot, err
}

// Hardlink creates a hardlink at the path
// source is the path to the file to link to
func (v *Fs) Hardlink(source, newname string, perm os.FileMode, modTime time.Time) (*Fs, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	paths, err := split(newname)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, ErrOutsideFilesystem
	}

	ln, _, err := v.fsFrom(source, -1)
	if err != nil {
		return nil, err
	}
	newRoot, err := v.hardlink(ln, paths, perm, modTime)
	return newRoot, err
}

// ---------------------Disk Operations--------------------

// Walk calls the callback for each file in the virtual filesystem
// if ErrDontWalk is returned, it will not walk the children
// the path walks the first value at that path, so if it has children:
// the path could be returned multiple times
func (v *Fs) Walk(path string, callback func(string, *Fs) error) error {
	toWalk, path, err := v.fsFrom(path, 0)
	if err != nil {
		return fmt.Errorf("%v: %v (walk)", err, path)
	}
	return toWalk.walkRecursive(path, callback)
}

// fsFrom returns the fileInfo for the path
// NOTE: at -1 returns that last one
func (v *Fs) fsFrom(path string, at int) (*Fs, string, error) {
	paths, err := split(path)
	if err != nil {
		return nil, "", err
	}
	// clean path
	path = filepath.Join("/", filepath.Join(paths...))

	toReturn, err := v.travelTo(paths, at)
	return toReturn, path, err
}

// ------------------split------------------
// split builds an array of paths from a string
// makes sure to remove "." and ".." from the path
// and returns an error if ".." is used to go outside the filesystem
func split(dir string) ([]string, error) {
	// cant use Clean b/c vulnerable to path traversal
	// dir := filepath.Clean(toClean)

	toReturn := []string{}
	for _, p := range strings.Split(dir, "/") {
		if p == "" || p == "." {
			continue
		} else if p == ".." {
			if len(toReturn) == 0 {
				return nil, ErrOutsideFilesystem
			}
			toReturn = toReturn[:len(toReturn)-1]
		} else {
			toReturn = append(toReturn, p)
		}
	}
	return toReturn, nil
}

//------------------split------------------

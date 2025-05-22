package virtualfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	// log "github.com/sirupsen/logrus"
)

// Close save the virtual file system to the disk
func (v *Fs) Close() error {
	if err := v.isClosed(); err != nil {
		return err
	}
	if err := v.isChild(); err != nil {
		return err
	}

	v.closed = true
	return v.save()
}

// isClosed checks if the virtual file system is closed
// (i.e. the db has been saved so dont add anything)
func (v *Fs) isClosed() error {
	if v.closed {
		return ErrClosed
	}
	return nil
}

func (v *Fs) isChild() error {
	if v.parent != nil {
		return ErrChild
	}
	return nil
}

// newFs Returns a new virtual filesystem from root
func (v *Fs) newFs(root *FileInfo) *Fs {
	return &Fs{
		root:   root,
		parent: v.parent,
	}
}

// FsFrom returns a virtual filesystem from the given path
func (v *Fs) FsFrom(path string) (*Fs, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	newRoot, _, err := v.fileInfoFrom(path, -1)
	if err != nil {
		return nil, fmt.Errorf("%v: %v (FsFrom)", err, path)
	}

	return v.newFs(newRoot), nil
}

// NewFsChild returns the virtual filesystem from the given path
// creates the directory (and parents) if it does not exist
func (v *Fs) NewFsChild(path string) (*Fs, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	paths, err := split(path)
	if err != nil || len(paths) != 1 {
		return nil, fmt.Errorf("error creating new child %v", path)
	}

	newRoot, err := v.root.mkdirP(paths, v.root.mode, v.root.modTime)
	if err != nil {
		return nil, err
	}

	return v.newFs(newRoot), nil
}

// --------Root stuff----------
// ProcessError returns an error if the filesystem has an error
func (v *Fs) ProcessError() error {
	if v.root.db.err {
		return ErrInFilesystem
	}
	return nil
}

// ProcessWarning returns an error if the filesystem has a warning
func (v *Fs) ProcessWarning() error {
	if v.root.db.warn {
		return ErrInFilesystem
	}
	return nil
}

// Root returns the FileInfo for the root of the filesystem
func (v *Fs) Root() *FileInfo {
	return v.root
}

//--------Root stuff----------

// ---------------------Disk Operations--------------------
// Stat returns an os.FileInfo like object for the path
func (v *Fs) Stat(path string) (*FileInfo, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	toWalk, _, err := v.fileInfoFrom(path, -1)
	if err != nil {
		return nil, err
	}
	return toWalk, nil
}

// StatAt returns an os.FileInfo like object for the path and index (if there are multiple, like for compression)
func (v *Fs) StatAt(path string, at int) (*FileInfo, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	toWalk, _, err := v.fileInfoFrom(path, at)
	if err != nil {
		return nil, err
	}
	return toWalk, nil
}

// Open returns an os.File for the path, if no path is given, it will return the root
func (v *Fs) Open(path string) (*os.File, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	toWalk, _, err := v.fileInfoFrom(path, -1)
	if err != nil {
		return nil, err
	}
	return toWalk.Open()
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
	newRoot, err := v.root.mkdirP(paths, perm, modTime)
	return v.newFs(newRoot), err
}

// TouchWithoutPath creates a child file
func (v *Fs) TouchWithoutPath(perm os.FileMode, modTime time.Time) (*Fs, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	newFileInfo, err := v.root.touch([]string{}, perm, modTime)
	return v.newFs(newFileInfo), err
}

// Touch creates a file to the path
func (v *Fs) Touch(path string, perm os.FileMode, modTime time.Time) (*Fs, error) {
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

	newFileInfo, err := v.root.touch(paths, perm, modTime)
	return v.newFs(newFileInfo), err
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

	newRoot, err := v.root.symlink(source, paths, perm, modTime)
	return v.newFs(newRoot), err
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

	ln, _, err := v.fileInfoFrom(source, -1)
	if err != nil {
		return nil, err
	}
	newRoot, err := v.root.hardlink(ln, paths, perm, modTime)
	return v.newFs(newRoot), err
}

// ---------------------Disk Operations--------------------

// Walk calls the callback for each file in the virtual filesystem
// if ErrDontWalk is returned, it will not walk the children
func (v *Fs) Walk(path string, callback func(string, *FileInfo) error) error {
	toWalk, path, err := v.fileInfoFrom(path, -1)
	if err != nil {
		return fmt.Errorf("%v: %v (walk)", err, path)
	}
	return toWalk.walkRecursive(path, callback)
}

// fileInfoFrom returns the fileInfo for the path
// NOTE: at -1 returns that last one
func (v *Fs) fileInfoFrom(path string, at int) (*FileInfo, string, error) {
	paths, err := split(path)
	if err != nil {
		return nil, "", err
	}
	// clean path
	path = filepath.Join("/", filepath.Join(paths...))

	toReturn, err := v.root.travelTo(paths, at)
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

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
	v.closed = true
	return v.save()
}

// checkClosed checks if the virtual file system is closed
// (i.e. the db has been saved so dont add anything)
func (v *Fs) checkClosed() error {
	if v.closed {
		return ErrClosed
	}
	return nil
}

// FsFrom returns a virtual filesystem from the given path
func (v *Fs) FsFrom(path string) (*Fs, error) {
	err := v.checkClosed()
	if err != nil {
		return nil, err
	}

	newRoot, _, err := v.fileInfoFrom(path, -1)
	if err != nil {
		return nil, fmt.Errorf("%v: %v (FsFrom)", err, path)
	}

	return &Fs{root: newRoot}, nil
}

// NewFsChild returns the virtual filesystem from the given path
// creates the directory (and parents) if it does not exist
func (v *Fs) NewFsChild(path string) (*Fs, error) {
	err := v.checkClosed()
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

	return &Fs{root: newRoot}, nil
}

// FsChildren returns the direct children of the filesystem
func (v *Fs) FsChildren() (toReturn []*Fs) {
	if v.root.ref.child != nil {
		toReturn = append(toReturn, &Fs{root: v.root.ref.child})
	}

	if v.root.ref.children == nil {
		return
	}

	for _, n := range v.root.ref.children {
		toReturn = append(toReturn, &Fs{root: n})
	}
	return
}

// --------Root stuff----------
// TODO: might can remove
func (v *Fs) ErrorId() error {
	return v.root.ErrorId()
}

// Error adds a error to the filesystem
func (v *Fs) Error(err error) {
	v.root.error(err)
}

// Warning adds a warning to the filesystem
func (v *Fs) Warning(err error) {
	v.root.warning(err)
}

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

// LocalPath returns the underlying path to this file in the virtual filesystem
// i.e. the unique file
func (v *Fs) LocalPath() string {
	return v.root.Path()
}

// TagS set a tag on this filesystem
// Note: its on the "reference" so same sha/data will have same tags
// mostly can be used for marking what extracter was used
func (v *Fs) TagS(key string, value any) {
	v.root.tagS(key, value)
}

// TagSIfBlank set a tag on this filesystem if its not been set yet,
// if it has it will raise an error. Multithreaded safe.
func (v *Fs) TagSIfBlank(key string, value any) error {
	return v.root.tagSIfBlank(key, value)
}

// TagG: Get a tag, returns false if not found
func (v *Fs) TagG(key string) (any, bool) {
	return v.root.tagG(key)
}

//--------Root stuff----------

// ---------------------Disk Operations--------------------
// Stat returns an os.FileInfo like object for the path
func (v *Fs) Stat(path string) (*FileInfo, error) {
	err := v.checkClosed()
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
	err := v.checkClosed()
	if err != nil {
		return nil, err
	}

	toWalk, _, err := v.fileInfoFrom(path, at)
	if err != nil {
		return nil, err
	}
	return toWalk, nil
}

// Open returns an os.File for the path
func (v *Fs) Open(path string) (*os.File, error) {
	err := v.checkClosed()
	if err != nil {
		return nil, err
	}

	toWalk, _, err := v.fileInfoFrom(path, -1)
	if err != nil {
		return nil, err
	}
	return toWalk.Open()
}

// Path returns the underlying path (for the path given) in the virtual filesystem
// i.e. the unique file
func (v *Fs) Path(path string) (string, error) {
	err := v.checkClosed()
	if err != nil {
		return "", err
	}

	toWalk, _, err := v.fileInfoFrom(path, -1)
	if err != nil {
		return "", err
	}
	return toWalk.Path(), nil
}

// MkdirP creates a directory and all the parents if they do not exist
func (v *Fs) MkdirP(path string, perm os.FileMode, modTime time.Time) error {
	err := v.checkClosed()
	if err != nil {
		return err
	}

	paths, err := split(path)
	if err != nil {
		return err
	}
	_, err = v.root.mkdirP(paths, perm, modTime)
	return err
}

// CreateChild creates a file under the root (i.e. for a compression that has a single
// file, not a path. Like gz vs tar.)
func (v *Fs) CreateChild(perm os.FileMode, modTime time.Time) (*myFile, error) {
	err := v.checkClosed()
	if err != nil {
		return nil, err
	}

	newFileInfo, err := v.root.touch([]string{}, perm, modTime)
	if err != nil {
		return nil, err
	}

	return newFileInfo.create()
}

// Create creates a file at the path
func (v *Fs) Create(path string, perm os.FileMode, modTime time.Time) (*myFile, error) {
	err := v.checkClosed()
	if err != nil {
		return nil, err
	}

	paths, err := split(path)
	if err != nil {
		return nil, err
	}
	if len(path) == 0 {
		panic("path shoudnt be empty")
	}

	newFileInfo, err := v.root.touch(paths, perm, modTime)
	if err != nil {
		return nil, err
	}

	return newFileInfo.create()
}

// Symlink creates a symlnk at the path
func (v *Fs) Symlink(oldname, newname string, perm os.FileMode, modTime time.Time) error {
	err := v.checkClosed()
	if err != nil {
		return err
	}

	paths, err := split(newname)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		panic("path shoudnt be empty")
	}

	_, err = v.root.symlink(oldname, paths, perm, modTime)
	return err
}

// ---------------------Disk Operations--------------------

// Walk calls the callback for each file in the virtual filesystem
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

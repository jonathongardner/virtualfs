package virtualfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jonathongardner/fifo/filetype"
	// log "github.com/sirupsen/logrus"
)

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
	newRoot, err := v.mkdirPRecursive(paths, perm, modTime)
	return newRoot, err
}

func (n *Fs) mkdirPRecursive(paths []string, perm os.FileMode, modTime time.Time) (*Fs, error) {
	// TODO: might want to think about permision of the child dir
	// This is important for something like a tar that first entry is `./`
	// that might have permissions on in that are different from the root
	// or maybe the gz file, can it have different permissions?
	if len(paths) == 0 {
		return n, nil
	}

	firstPath, paths := paths[0], paths[1:]

	// return err if `child` not `children`
	child, err := n.ref.getChildren(firstPath)
	if err != nil {
		return nil, err
	}
	if child == nil || child.ref.typ != filetype.Dir {
		// NOTE:orphan if the child is not a directory then we could orphan a file if its not used anywhere else
		child, err = n.ref.setChildren(n.newFs(firstPath, perm, modTime).setToDir())
		// shouldnt happen since `getChild` would have returned error first but in case logic changes in setChild
		if err != nil {
			return nil, err
		}
	}
	return child.mkdirPRecursive(paths, perm, modTime)
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

	newRoot, err := v.symlinkRecursive(source, paths, perm, modTime)
	return newRoot, err
}

func (n *Fs) symlinkRecursive(linkname string, paths []string, perm os.FileMode, modTime time.Time) (*Fs, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("Symlink needs a path to set")
	}

	last := len(paths) - 1
	paths, name := paths[:last], paths[last]

	dir, err := n.mkdirPRecursive(paths, perm, modTime)
	if err != nil {
		return nil, err
	}
	// NOTE: orphan this could orphin some references, might want to clean up if reference is not needed
	return dir.ref.setChildren(n.newFs(name, perm, modTime).setToSym(linkname))
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
	newRoot, err := v.hardlinkRecursive(ln, paths, perm, modTime)
	return newRoot, err
}

// hardlink creates a hardlink at the path
// ln is the file to link to
func (n *Fs) hardlinkRecursive(ln *Fs, paths []string, perm os.FileMode, modTime time.Time) (*Fs, error) {
	if len(paths) == 0 {
		// NOTE: orphan this could orphin some references, might want to clean up if reference is not needed
		return n.ref.setChild(n.newFs(n.name, perm, modTime))
	}
	last := len(paths) - 1
	paths, name := paths[:last], paths[last]

	dir, err := n.mkdirPRecursive(paths, perm, modTime)
	if err != nil {
		return nil, err
	}
	// NOTE: orphan this could orphin some references, might want to clean up if reference is not needed
	return dir.ref.setChildren(n.newFsWithReference(name, perm, modTime, ln.ref))
}

// CreateWithoutPath creates a child file
func (v *Fs) CreateWithoutPath(perm os.FileMode, modTime time.Time) (*Fs, error) {
	err := v.isClosed()
	if err != nil {
		return nil, err
	}

	newFileInfo, err := v.createRecursive([]string{}, perm, modTime)
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

	newFileInfo, err := v.createRecursive(paths, perm, modTime)
	return newFileInfo, err
}

func (n *Fs) createRecursive(paths []string, perm os.FileMode, modTime time.Time) (*Fs, error) {
	if len(paths) == 0 {
		// NOTE: orphan this could orphin some references, might want to clean up if reference is not needed
		return n.ref.setChild(n.newFs(n.name, perm, modTime))
	}
	last := len(paths) - 1
	paths, name := paths[:last], paths[last]

	dir, err := n.mkdirPRecursive(paths, perm, modTime)
	if err != nil {
		return nil, err
	}
	// NOTE: orphan this could orphin some references, might want to clean up if reference is not needed
	return dir.ref.setChildren(n.newFs(name, perm, modTime))
}

// -------------------------File------------------------
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

// -------------------------File------------------------

// ---------------------Travel Operations--------------------
// Walk calls the callback for each file in the virtual filesystem
// if ErrDontWalk is returned, it will not walk the children
// the path walks the first value at that path, so if it has children:
// the path could be returned multiple times
func (v *Fs) Walk(path string, callback func(string, *Fs) error) error {
	toWalk, path, err := v.fsFrom(path, 0)
	if err != nil {
		return fmt.Errorf("%w: %v (walk)", err, path)
	}
	return toWalk.walkRecursive(path, false, func(path string, child bool, fs *Fs) error { return callback(path, fs) })
}

// walkRecursive is a recursive function that walks the filesystem used by Walk and seralize
func (n *Fs) walkRecursive(path string, child bool, callback func(string, bool, *Fs) error) error {
	err := callback(path, child, n)
	if errors.Is(err, ErrDontWalk) {
		return nil
	}
	if err != nil {
		return err
	}

	if n.ref.child != nil {
		if err := n.ref.child.walkRecursive(path, true, callback); err != nil {
			return err
		}
	} else if len(n.ref.children) > 0 {
		names := make([]string, 0, len(n.ref.children))
		for n := range n.ref.children {
			names = append(names, n)
		}
		slices.Sort(names)
		for _, name := range names {
			if err := n.ref.children[name].walkRecursive(filepath.Join(path, name), false, callback); err != nil {
				return err
			}
		}
	}

	return nil
}

func (n *Fs) travelTo(paths []string, at int) (*Fs, error) {
	// we want to return the last child i.e. if ask for path `/foo/bar.gz` we need to get the extracted `gz` file
	if n.ref.child != nil && at != 0 {
		if len(paths) == 0 {
			at = at - 1
		}
		return n.ref.child.travelTo(paths, at)
	}
	if len(paths) == 0 {
		if at > 0 {
			// still return last found
			return n, ErrNotFound
		}

		return n, nil
	}

	toWalk, ok := n.ref.children[paths[0]]
	if ok {
		return toWalk.travelTo(paths[1:], at)
	}
	return n, ErrNotFound
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

// ---------------------Travel Operations--------------------

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

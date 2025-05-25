package virtualfs

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/jonathongardner/fifo/filetype"
)

// ---------------------Disk Operations--------------------

func (n *Fs) mkdirP(paths []string, perm os.FileMode, modTime time.Time) (*Fs, error) {
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
		child, err = n.ref.setChildren(newFileInfo(n.db, firstPath, perm, modTime).setToDir())
		// shouldnt happen since `getChild` would have returned error first but in case logic changes in setChild
		if err != nil {
			return nil, err
		}
	}
	return child.mkdirP(paths, perm, modTime)
}

func (n *Fs) touch(paths []string, perm os.FileMode, modTime time.Time) (*Fs, error) {
	if len(paths) == 0 {
		// NOTE: orphan this could orphin some references, might want to clean up if reference is not needed
		return n.ref.setChild(newFileInfo(n.db, n.name, perm, modTime))
	}
	last := len(paths) - 1
	paths, name := paths[:last], paths[last]

	dir, err := n.mkdirP(paths, perm, modTime)
	if err != nil {
		return nil, err
	}
	// NOTE: orphan this could orphin some references, might want to clean up if reference is not needed
	return dir.ref.setChildren(newFileInfo(n.db, name, perm, modTime))
}

func (n *Fs) symlink(linkname string, paths []string, perm os.FileMode, modTime time.Time) (*Fs, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("Symlink needs a path to set")
	}

	last := len(paths) - 1
	paths, name := paths[:last], paths[last]

	dir, err := n.mkdirP(paths, perm, modTime)
	if err != nil {
		return nil, err
	}
	// NOTE: orphan this could orphin some references, might want to clean up if reference is not needed
	return dir.ref.setChildren(newFileInfo(n.db, name, perm, modTime).setToSym(linkname))
}

// ln is the file to link to
func (n *Fs) hardlink(ln *Fs, paths []string, perm os.FileMode, modTime time.Time) (*Fs, error) {
	if len(paths) == 0 {
		// NOTE: orphan this could orphin some references, might want to clean up if reference is not needed
		return n.ref.setChild(newFileInfo(n.db, n.name, perm, modTime))
	}
	last := len(paths) - 1
	paths, name := paths[:last], paths[last]

	dir, err := n.mkdirP(paths, perm, modTime)
	if err != nil {
		return nil, err
	}
	// NOTE: orphan this could orphin some references, might want to clean up if reference is not needed
	return dir.ref.setChildren(newFileInfoWithReference(n.db, name, perm, modTime, ln.ref))
}

// ---------------------Disk Operations--------------------

func (n *Fs) walkRecursive(path string, callback func(string, *Fs) error) error {
	err := callback(path, n)
	if err == ErrDontWalk {
		return nil
	}
	if err != nil {
		return err
	}

	if n.ref.child != nil {
		if err := n.ref.child.walkRecursive(path, callback); err != nil {
			return err
		}
	} else if len(n.ref.children) > 0 {
		names := make([]string, 0, len(n.ref.children))
		for n := range n.ref.children {
			names = append(names, n)
		}
		slices.Sort(names)
		for _, name := range names {
			if err := n.ref.children[name].walkRecursive(filepath.Join(path, name), callback); err != nil {
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
			return nil, ErrNotFound
		}

		return n, nil
	}

	toWalk, ok := n.ref.children[paths[0]]
	if ok {
		return toWalk.travelTo(paths[1:], at)
	}
	return nil, ErrNotFound
}

// ---------------------FileInfo Methods--------------------
func (fi *Fs) Name() string {
	return fi.name
}
func (fi *Fs) Size() int64 {
	return fi.ref.size
}
func (fi *Fs) Mode() os.FileMode {
	return fi.mode
}
func (fi *Fs) ModTime() time.Time {
	return fi.modTime
}
func (fi *Fs) IsDir() bool {
	return fi.mode.IsDir()
}
func (fi *Fs) Sys() any {
	return nil
}

// ---------------------FileInfo Methods--------------------

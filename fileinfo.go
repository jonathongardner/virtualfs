package virtualfs

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jonathongardner/virtualfs/filetype"
)

// -------------------Node---------------------
// unique to the file like mode path/name, etc
type FileInfo struct {
	db          *referenceDB
	name        string
	mode        os.FileMode
	symlinkPath string
	modTime     time.Time
	ref         *reference
}

func newFileInfo(db *referenceDB, name string, mode os.FileMode, modTime time.Time) *FileInfo {
	processed := &atomic.Bool{}
	processed.Store(false)
	reference := &reference{id: uuid.New().String(), processed: processed, children: make(map[string]*FileInfo)}
	return &FileInfo{db: db, name: name, mode: mode, modTime: modTime, ref: reference}
}

func newDirFileInfo(db *referenceDB, name string, mode os.FileMode, modTime time.Time) *FileInfo {
	toReturn := newFileInfo(db, name, mode, modTime)
	toReturn.ref.typ = filetype.Dir
	return toReturn
}

func newSymlinkFileInfo(db *referenceDB, oldname, name string, mode os.FileMode, modTime time.Time) *FileInfo {
	toReturn := newFileInfo(db, name, mode, modTime)
	toReturn.ref.typ = filetype.Symlink
	toReturn.symlinkPath = oldname
	return toReturn
}

// ---------------------Disk Operations--------------------
func (n *FileInfo) Open() (*os.File, error) {
	if n.ref.typ == filetype.Symlink {
		return nil, fmt.Errorf("cannot open a symlink")
	}
	if n.ref.typ == filetype.Dir {
		return nil, fmt.Errorf("cannot open a directory")
	}

	return n.ref.open(n.db.storageDir)
}

func (n *FileInfo) mkdirP(paths []string, perm os.FileMode, modTime time.Time) (*FileInfo, error) {
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
		child, err = n.ref.setChildren(newDirFileInfo(n.db, firstPath, perm, modTime))
		// shouldnt happen since `getChild` would have returned error first but in case logic changes in setChild
		if err != nil {
			return nil, err
		}
	}
	return child.mkdirP(paths, perm, modTime)
}

func (n *FileInfo) touch(paths []string, perm os.FileMode, modTime time.Time) (*FileInfo, error) {
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

func (n *FileInfo) create() (*myWriteCloser, error) {
	file, err := n.ref.create(n.db.storageDir)
	if err != nil {
		return nil, err
	}
	return createMyWriterCloser(n, file), nil
}

func (n *FileInfo) symlink(oldname string, paths []string, perm os.FileMode, modTime time.Time) (*FileInfo, error) {
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
	return dir.ref.setChildren(newSymlinkFileInfo(n.db, oldname, name, perm, modTime))
}

// ---------------------Disk Operations--------------------

func (n *FileInfo) walkRecursive(path string, callback func(string, *FileInfo) error) error {
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

func (n *FileInfo) travelTo(paths []string, at int) (*FileInfo, error) {
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
func (fi *FileInfo) Name() string {
	return fi.name
}
func (fi *FileInfo) Size() int64 {
	return fi.ref.size
}
func (fi *FileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi *FileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi *FileInfo) IsDir() bool {
	return fi.mode.IsDir()
}
func (fi *FileInfo) Sys() any {
	return nil
}

// ---------------------FileInfo Methods--------------------
func (fi *FileInfo) Filetype() filetype.Filetype {
	return fi.ref.typ
}

func (fi *FileInfo) ErrorId() error {
	return fmt.Errorf("id: %v, name: %v, type: %v,", fi.ref.id, fi.name, fi.ref.typ)
}

// -----------------------Checksums--------------------------
func (fi *FileInfo) Sha512() string {
	return fi.ref.sha512
}

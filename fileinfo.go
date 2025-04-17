package virtualfs

import (
	// "compress/gzip"

	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/jonathongardner/fifo"
	"github.com/jonathongardner/virtualfs/filetype"
	// mgzip "github.com/jonathongardner/virtualfs/gzip"
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

// newFileInfo creates a new file info object, return error if read only
func newFileInfo(db *referenceDB, name string, mode os.FileMode, modTime time.Time) *FileInfo {
	return &FileInfo{db: db, name: name, mode: mode, modTime: modTime, ref: newReference()}
}
func (n *FileInfo) setToDir() *FileInfo {
	n.mode |= fs.ModeDir
	n.ref.typ = filetype.Dir
	return n
}
func (n *FileInfo) setToSym(oldname string) *FileInfo {
	n.mode |= fs.ModeSymlink
	n.ref.typ = filetype.Symlink
	n.symlinkPath = oldname
	return n
}
func (n *FileInfo) error(err error) {
	n.ref.err = err
	n.db.err = true
}
func (n *FileInfo) warning(warn error) {
	n.ref.warn = append(n.ref.warn, warn)
	n.db.warn = true
}
func (n *FileInfo) TagS(key string, value any) {
	n.ref.tags.Store(key, value)
}
func (n *FileInfo) TagSIfBlank(key string, value any) error {
	_, loaded := n.ref.tags.LoadOrStore(key, value)
	if loaded {
		return ErrAlreadyExist
	}
	return nil
}
func (n *FileInfo) TagG(key string) (any, bool) {
	return n.ref.tags.Load(key)
}
func (n *FileInfo) TagD(key string) {
	n.ref.tags.Delete(key)
}

// ---------------------Disk Operations--------------------
func (n *FileInfo) Open() (fifo.ReadCloseReseter, error) {
	// TODO: match how os would handle this
	if n.ref.typ == filetype.Symlink {
		return nil, fmt.Errorf("cannot open a symlink")
	}
	if n.ref.typ == filetype.Dir {
		return nil, fmt.Errorf("cannot open a directory")
	}

	// TODO: should bound the wf to now allow going backwards
	r, err := n.db.bfp.Get(n.ref.offset, n.ref.size)
	if err != nil {
		return nil, fmt.Errorf("error getting file %v", err)
	}
	return r, nil
	// return mgzip.NewReader(r)
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
		child, err = n.ref.setChildren(newFileInfo(n.db, firstPath, perm, modTime).setToDir())
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

func (n *FileInfo) copy(r io.Reader) error {
	mw := newMultiWriter(io.Discard)
	if _, err := io.Copy(mw, r); err != nil {
		return fmt.Errorf("error getting node info: %v", err)
	}

	err := n.ref.build(mw)
	if err != nil {
		return err
	}

	var set bool
	n.ref, set = n.db.setIfEmpty(n.ref)
	// We have already seen this sha512 so we dont need to copy the data to the file
	if !set {
		return nil
	}

	// If we havent seen this file before, check if we can rewind to write it
	// if we cant then raise an error and client can handle it
	s, ok := r.(io.Seeker)
	if !ok {
		return ErrCantWriteNewFile
	}
	s.Seek(0, io.SeekStart)

	// TODO: can maybe handle a cached file here
	return n.Copy(r)
}

// If the node values are not set, set them
// If the node values are set, compare them
func (n *FileInfo) Copy(r io.Reader) error {
	var callback func(file *os.File) (int64, error)

	if n.ref.sha512 == "" {
		callback = func(file *os.File) (int64, error) {
			// Write to file and get shas, etc
			// comp := gzip.NewWriter(file)
			// defer comp.Close()
			// mw := newMultiWriter(comp)
			mw := newMultiWriter(file)
			size, err := io.Copy(mw, r)
			if err != nil {
				return size, fmt.Errorf("error copying to file: %v", err)
			}

			// set shas, etc on node
			err = n.ref.build(mw)
			if err != nil {
				return size, err
			}

			// check if we have already seen this sha512
			var set bool
			n.ref, set = n.db.setIfEmpty(n.ref)
			if set {
				return size, nil
			}
			// if we have seen before we need rollback
			_, err = file.Seek(-1*size, io.SeekEnd)
			return 0, err
		}
	} else {
		callback = func(file *os.File) (int64, error) {
			// Write to file and get shas, etc
			sha512v := sha512.New()
			// comp := gzip.NewWriter(file)
			// defer comp.Close()
			// mw := io.MultiWriter(sha512v, comp)
			mw := io.MultiWriter(sha512v, file)
			size, err := io.Copy(mw, r)
			if err != nil {
				return size, fmt.Errorf("error copying to file: %v", err)
			}

			// if the copy sha512 does not match the one we have then we have a problem
			if n.ref.sha512 != hex.EncodeToString(sha512v.Sum(nil)) {
				return size, fmt.Errorf("hash mismatch on adding file")
			}

			return size, nil
		}

	}

	if err := n.db.add(n.ref, callback); err != nil {
		return err
	}
	return nil
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
	return dir.ref.setChildren(newFileInfo(n.db, name, perm, modTime).setToSym(oldname))
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
	// return fi.ref.tags
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

package virtualfs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jonathongardner/fifo/filetype"
	// log "github.com/sirupsen/logrus"
)

type Fs struct {
	// fs
	db     *referenceDB
	isRoot bool
	closed bool
	// file info (unique to the location)
	name        string
	mode        os.FileMode
	symlinkPath string
	modTime     time.Time
	// unique to file (checksums, filetype, etc)
	ref *reference
}

// NewFsFromDb loads a virtual file system from a directory DB
func NewFsFromDb(storageDir string) (*Fs, error) {
	toReturn := &Fs{isRoot: true}
	return toReturn, toReturn.load(storageDir)
}

// NewFs creates a new virtual file system from a file or stdin
func NewFs(storageDir, name string, mode os.FileMode, modTime time.Time, r io.Reader) (*Fs, error) {
	err := os.Mkdir(storageDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to create storage dir: %w", err)
	}

	fs := &Fs{
		isRoot:  true,
		db:      newReferenceDB(storageDir),
		name:    name,
		mode:    mode,
		modTime: modTime,
		ref: &reference{
			id:       uuid.New().String(),
			children: make(map[string]*Fs),
		},
	}
	if mode.IsDir() {
		fs.setToDir()
	} else {
		if err := fs.copyReader(r); err != nil {
			return nil, fmt.Errorf("couldn't copy from reader (%v) - %w", name, err)
		}
	}

	return fs, nil
}

// ----------------Helpers--------------------
// newFsWithReference creates a new file info object
func (f *Fs) newFsWithReference(name string, mode os.FileMode, modTime time.Time, reference *reference) *Fs {
	return &Fs{db: f.db, name: name, mode: mode, modTime: modTime, isRoot: false, ref: reference}
}

// newFs creates a new file info object and reference
func (f *Fs) newFs(name string, mode os.FileMode, modTime time.Time) *Fs {
	return f.newFsWithReference(
		name,
		mode,
		modTime,
		&reference{
			id:       uuid.New().String(),
			children: make(map[string]*Fs),
		},
	)
}

// setToDir sets the fs to a directory and mode
func (n *Fs) setToDir() *Fs {
	n.mode |= fs.ModeDir
	n.ref.typ = filetype.Dir
	return n
}

// setToSym sets the fs to a symlink and mode and set symlinkPath
func (n *Fs) setToSym(oldname string) *Fs {
	n.mode |= fs.ModeSymlink
	n.ref.typ = filetype.Symlink
	n.symlinkPath = oldname
	return n
}

// updateIfDuplicateRef if sha512 already seen it will use that ref and return true
// if its the first time we see the sha512 then dont change anything and return false
func (n *Fs) updateIfDuplicateRef() (updated bool, err error) {
	oldRef := n.ref

	n.ref, updated = n.db.updateIfDuplicate(n.ref)
	if updated {
		err = n.checkIfCircular(n.ref.sha512, true)
		if err != nil {
			n.ref = oldRef // revert to old reference
			updated = false
		}
	}
	return
}

func (n *Fs) checkIfCircular(sha512 string, ignore bool) error {
	if !ignore && n.ref.sha512 == sha512 {
		return ErrCircularReference
	}

	for _, child := range n.ref.children {
		if err := child.checkIfCircular(sha512, false); err != nil {
			return err
		}
	}
	if n.ref.child != nil {
		return n.ref.child.checkIfCircular(sha512, false)
	}
	return nil
}

// ----------------Helpers--------------------

// copyReader copies the contents of a reader to the virtual filesystem
func (fs *Fs) copyReader(src io.Reader) error {
	newFile, err := fs.CreateFile()
	if err != nil {
		return err
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, src)
	if err != nil {
		return err
	}

	return nil
}

// Close save the virtual file system to the disk
func (v *Fs) Close() error {
	if err := v.isClosed(); err != nil {
		return err
	}
	if !v.IsRoot() {
		return ErrChild
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

// IsRoot checks if the virtual file system is the root
func (v *Fs) IsRoot() bool {
	return v.isRoot
}

// IsCompression check if the file has `child` or `children`
// if its `child` then its a compressed file
func (v *Fs) IsCompression() bool {
	return v.ref.child != nil
}

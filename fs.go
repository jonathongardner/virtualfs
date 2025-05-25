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
	parent *Fs
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
	toReturn := &Fs{}
	return toReturn, toReturn.load(storageDir)
}

// NewFs creates a new virtual file system from a file or stdin
func NewFs(storageDir, name string, mode os.FileMode, modTime time.Time, r io.Reader) (*Fs, error) {
	err := os.Mkdir(storageDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to create storage dir: %v", err)
	}

	fs := newFileInfo(newReferenceDB(storageDir), name, mode, modTime)
	if mode.IsDir() {
		fs.setToDir()
	} else {
		if err := fs.copyReader(r); err != nil {
			return nil, fmt.Errorf("couldn't copy from reader (%v) - %v", name, err)
		}
	}

	return fs, nil
}

// ----------------Helpers--------------------
// newFileInfoWithReference creates a new file info object
func newFileInfoWithReference(db *referenceDB, name string, mode os.FileMode, modTime time.Time, reference *reference) *Fs {
	return &Fs{db: db, name: name, mode: mode, modTime: modTime, ref: reference}
}

// newFileInfo creates a new file info object and reference
func newFileInfo(db *referenceDB, name string, mode os.FileMode, modTime time.Time) *Fs {
	return newFileInfoWithReference(
		db,
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
func (n *Fs) updateIfDuplicateRef() bool {
	var set bool
	n.ref, set = n.db.setIfEmpty(n.ref)
	// if its no new than it was updated
	return !set
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

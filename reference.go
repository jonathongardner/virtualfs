package virtualfs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/jonathongardner/virtualfs/filetype"
)

// -------------------Reference---------------------
var ErrAlreadyProcessed = fmt.Errorf("reference already extracted")
var ErrAlreadyHasChild = fmt.Errorf("has child")
var ErrAlreadyHasChildren = fmt.Errorf("has children")

// everything unique to a file (i.e not mode or name)
type reference struct {
	id        string
	size      int64
	typ       filetype.Filetype
	md5       string
	sha1      string
	sha256    string
	sha512    string
	entropy   float64
	processed *atomic.Bool
	// SymlinkPath string            `json:"symlinkPath"`
	// Archive     bool              `json:"archive"`
	child    *FileInfo
	children map[string]*FileInfo
}

func (r *reference) storagePath(storageDir string) string {
	return filepath.Join(storageDir, r.id)
}

func (r *reference) remove(storageDir string) error {
	return os.Remove(r.storagePath(storageDir))
}

func (r *reference) create(storageDir string) (*os.File, error) {
	return os.Create(r.storagePath(storageDir))
}
func (r *reference) open(storageDir string) (*os.File, error) {
	return os.Open(r.storagePath(storageDir))
}

// Return old value, if old valud is true then it was already extracted
// might should return an error for that?
func (r *reference) process() error {
	if r.processed.Swap(true) {
		return ErrAlreadyProcessed
	}
	return nil
}
func (r *reference) getChildren(name string) (*FileInfo, error) {
	if r.child != nil {
		return nil, fmt.Errorf("has child not children %v", name)
	}
	return r.children[name], nil
}
func (r *reference) setChildren(child *FileInfo) (*FileInfo, error) {
	if r.child != nil {
		return nil, ErrAlreadyHasChild
	}
	r.children[child.name] = child
	return child, nil
}
func (r *reference) setChild(child *FileInfo) (*FileInfo, error) {
	if len(r.children) != 0 {
		return nil, ErrAlreadyHasChildren
	}
	r.child = child
	return child, nil
}
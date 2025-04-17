package virtualfs

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/jonathongardner/virtualfs/filetype"
)

// -------------------Reference---------------------
var ErrAlreadyExist = fmt.Errorf("reference already exist")
var ErrAlreadyHasChild = fmt.Errorf("has child")
var ErrAlreadyHasChildren = fmt.Errorf("has children")

// everything unique to a file (i.e not mode or name)
type reference struct {
	id             string // might can remove
	offset         int64
	compressedSize int64
	// Added by copy
	size    int64
	typ     filetype.Filetype
	md5     string
	sha1    string
	sha256  string
	sha512  string
	entropy float64
	// Added by copy
	err      error
	warn     []error
	tags     sync.Map
	child    *FileInfo
	children map[string]*FileInfo
}

func newReference() *reference {
	return &reference{
		id:       uuid.New().String(),
		children: make(map[string]*FileInfo),
		offset:   -1,
	}
}

func (ref *reference) build(mw *multiWriter) error {
	if ref.sha512 != "" || ref.child != nil || len(ref.children) != 0 {
		return fmt.Errorf("already built")
	}

	ref.typ = mw.Value("filetype").(filetype.Filetype)
	ref.entropy = mw.Value("entropy").(float64)
	ref.md5 = mw.Value("md5").(string)
	ref.sha1 = mw.Value("sha1").(string)
	ref.sha256 = mw.Value("sha256").(string)
	ref.sha512 = mw.Value("sha512").(string)
	ref.size = mw.Value("size").(int64)

	return nil
}

// Return old value, if old value is true then it was already extracted
// might should return an error for that?
func (r *reference) getChildren(name string) (*FileInfo, error) {
	if r.child != nil {
		return nil, fmt.Errorf("has child not children %v", name)
	}
	return r.children[name], nil
}
func (r *reference) setChildren(child *FileInfo) (*FileInfo, error) {
	if child.db.wf == nil {
		return nil, ErrReadOnly
	}
	if r.child != nil {
		return nil, ErrAlreadyHasChild
	}
	r.children[child.name] = child
	return child, nil
}
func (r *reference) setChild(child *FileInfo) (*FileInfo, error) {
	if child.db.wf == nil {
		return nil, ErrReadOnly
	}
	if len(r.children) != 0 {
		return nil, ErrAlreadyHasChildren
	}
	r.child = child
	return child, nil
}

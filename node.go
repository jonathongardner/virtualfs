package virtualfs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jonathongardner/virtualfs/filetype"
)

// -------------------Reference---------------------
var ErrAlreadyProcessed = fmt.Errorf("reference already extracted")

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
	child    *node
	children map[string]*node
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

// -------------------Node---------------------
// unique to the file like mode path/name, etc
type node struct {
	name    string
	mode    os.FileMode
	modTime time.Time // TODO: handle
	ref     *reference
}

func newNode(name string, mode os.FileMode) *node {
	processed := &atomic.Bool{}
	processed.Store(false)
	reference := &reference{id: uuid.New().String(), processed: processed, children: make(map[string]*node)}
	return &node{name: name, mode: mode, ref: reference}
}

func newDirNode(name string, mode os.FileMode) *node {
	toReturn := newNode(name, mode)
	toReturn.ref.typ = filetype.Dir
	return toReturn
}

func (n *node) errorID() error {
	return fmt.Errorf("id: %v, name: %v, type: %v,", n.ref.id, n.name, n.ref.typ)
}

func (n *node) create(storageDir, name string, perm os.FileMode) (*node, *os.File, error) {
	// NOTE:orphan this could orphin some references, might want to clean up if reference is not needed
	newNode := newNode(name, perm)
	n.ref.children[name] = newNode
	file, err := newNode.ref.create(storageDir)

	return newNode, file, err
}
func (n *node) open(storageDir string) (*os.File, error) {
	return n.ref.open(storageDir)
}

func (n *node) mkdirP(paths []string, perm os.FileMode) (*node, error) {
	if len(paths) == 0 {
		return n, nil
	}

	firstPath, paths := paths[0], paths[1:]

	child, ok := n.ref.children[firstPath]
	if !ok {
		if n.ref.child != nil {
			return nil, fmt.Errorf("cannot mkdir %v, %v has a child", firstPath, n.ref.child.name)
		}
		n.ref.children[firstPath] = newDirNode(firstPath, perm)
		child = n.ref.children[firstPath]
	} else if child.ref.typ != filetype.Dir {
		// NOTE:orphan some references, only issue is it could be a possible large file
		// that isnt accessible so not needed and we could delete to free space
		n.ref.children[firstPath] = newDirNode(firstPath, perm)
		child = n.ref.children[firstPath]
	}
	return child.mkdirP(paths, perm)
}

func (n *node) walkRecursive(path string, callback func(string, os.FileInfo) error) error {
	err := callback(path, n)
	if err == ErrDontWalk {
		return nil
	}
	if err != nil {
		return err
	}

	if n.ref.children != nil {
		for name, child := range n.ref.children {
			if err := child.walkRecursive(filepath.Join(path, name), callback); err != nil {
				return err
			}
		}
	}
	if n.ref.child != nil {
		if err := n.ref.child.walkRecursive(path, callback); err != nil {
			return err
		}
	}

	return nil
}

func (n *node) MarshalJSON() ([]byte, error) {
	toReturn := make(map[string]any)
	toReturn["name"] = n.name
	toReturn["processed"] = n.ref.processed.Load()
	toReturn["size"] = n.ref.size
	toReturn["mode"] = n.mode
	toReturn["modTime"] = n.modTime
	toReturn["md5"] = n.ref.md5
	toReturn["sha1"] = n.ref.sha1
	toReturn["sha256"] = n.ref.sha256
	toReturn["sha512"] = n.ref.sha512
	toReturn["entropy"] = n.ref.entropy
	toReturn["type"] = n.ref.typ
	return json.Marshal(toReturn)
}

func (n *node) Name() string {
	return n.name
}
func (n *node) Size() int64 {
	return n.ref.size
}
func (n *node) Mode() os.FileMode {
	return n.mode
}
func (n *node) ModTime() time.Time {
	return n.modTime
}
func (n *node) IsDir() bool {
	return n.mode.IsDir()
}
func (n *node) Sys() any {
	return n
}

// ------------------node------------------

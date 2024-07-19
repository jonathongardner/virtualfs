package virtualfs

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jonathongardner/virtualfs/entropy"
	"github.com/jonathongardner/virtualfs/filetype"
	// log "github.com/sirupsen/logrus"
)

type Fs struct {
	storageDir string
	root       *node
	db         *referenceDB
}

// var root = &Entry{type: filetype.Directory}
var ErrDontWalk = fmt.Errorf("dont walk entries children")
var ErrNotFound = fmt.Errorf("file not found") // https://smyrman.medium.com/writing-constant-errors-with-go-1-13-10c4191617

func NewFs(storageDir, rootPath string) (*Fs, error) {
	reader := os.Stdin
	mode := os.FileMode(0755)
	filename := "stdin"

	if rootPath != "" {
		filename = path.Base(rootPath)
		fileToCopy, err := os.Open(rootPath)
		if err != nil {
			return nil, fmt.Errorf("couldn't open path (%v) - %v", rootPath, err)
		}
		defer fileToCopy.Close()

		fileInfo, err := fileToCopy.Stat()
		if err != nil {
			return nil, fmt.Errorf("couldn't get path info (%v) - %v", rootPath, err)
		}
		// TODO: in the future we could allow this and just copy everything in directory but for now this is fine
		if fileInfo.IsDir() {
			return nil, fmt.Errorf("must provide a file (not a directory)")
		}
		mode = fileInfo.Mode()
		reader = fileToCopy
	}
	rootNode := newNode(filename, mode)

	toReturn := &Fs{storageDir: storageDir, root: rootNode, db: newReferenceDB()}

	file, err := rootNode.ref.create(storageDir)
	if err != nil {
		return nil, err
	}

	writer := createMyWriterCloser(toReturn, rootNode, file)
	defer writer.Close()
	_, err = io.Copy(writer, reader)
	if err != nil {
		return nil, fmt.Errorf("couldn't copy file (%v) - %v", rootPath, err)
	}

	return toReturn, nil
}

func (v *Fs) FsFrom(path string) (*Fs, error) {
	toWalk, _, err := v.nodeFrom(path)
	if err != nil {
		return nil, fmt.Errorf("%v: %v (walk)", err, path)
	}
	return &Fs{storageDir: v.storageDir, root: toWalk, db: v.db}, nil
}

func (v *Fs) FsChildren() (toReturn []*Fs) {
	if v.root.ref.children == nil {
		return
	}

	for _, n := range v.root.ref.children {
		toReturn = append(toReturn, &Fs{storageDir: v.storageDir, root: n, db: v.db})
	}
	return
}

func (v *Fs) Process() error {
	return v.root.ref.process()
}

func (v *Fs) RootFiletype() filetype.Filetype {
	return v.root.ref.typ
}

func (v *Fs) RootMode() os.FileMode {
	return v.root.mode
}

func (v *Fs) RootOpen() (*os.File, error) {
	return v.root.open(v.storageDir)
}

func (v *Fs) RootErrorID() error {
	return v.root.errorID()
}

func (v *Fs) MkdirP(path string, perm os.FileMode) error {
	paths := split(path)
	// TODO: might want to think about permision of the child dir
	// This is important for something like a tar that first entry is `./`
	// that might have permissions on in that are different from the root
	if len(paths) == 0 {
		return nil // just return if trying to change root directory
	}

	_, err := v.root.mkdirP(paths, perm)
	return err
}

func (v *Fs) Create(path string, perm os.FileMode) (*myWriteCloser, error) {
	paths := split(path)
	if len(paths) == 0 {
		return nil, fmt.Errorf("path must be a directory")
	}
	last := len(paths) - 1
	paths, fileName := paths[:last], paths[last]

	node, err := v.root.mkdirP(paths, perm)
	if err != nil {
		return nil, err
	}

	newNode, file, err := node.create(v.storageDir, fileName, perm)
	if err != nil {
		return nil, err
	}
	return createMyWriterCloser(v, newNode, file), nil
}

func (v *Fs) Walk(path string, callback func(string, os.FileInfo) error) error {
	toWalk, path, err := v.nodeFrom(path)
	if err != nil {
		return fmt.Errorf("%v: %v (walk)", err, path)
	}
	return toWalk.walkRecursive(path, callback)
}

func (v *Fs) nodeFrom(path string) (*node, string, error) {
	toWalk := v.root
	paths := split(path)
	// clean paths
	path = filepath.Join("/", filepath.Join(paths...))

	for _, p := range paths {
		var ok bool
		toWalk, ok = toWalk.ref.children[p]
		if !ok {
			return nil, path, ErrNotFound
		}
	}
	return toWalk, path, nil
}

// ------------------split------------------
func split(dir string) (toReturn []string) {
	dir = filepath.Clean(dir)
	for _, p := range strings.Split(dir, "/") {
		if p != "" && p != "." {
			toReturn = append(toReturn, p)
		}
	}
	return
}

//------------------split------------------

// ------------------Writer Closer------------------
type myWriteCloser struct {
	md5     hash.Hash
	sha1    hash.Hash
	sha256  hash.Hash
	sha512  hash.Hash
	entropy *entropy.Writer
	size    int64
	ftype   *filetype.FiletypeWriter
	dst     io.WriteCloser
	mlt     io.Writer
	node    *node
	fs      *Fs
}

func createMyWriterCloser(fs *Fs, node *node, dst io.WriteCloser) *myWriteCloser {
	toReturn := &myWriteCloser{
		md5:     md5.New(),
		sha1:    sha1.New(),
		sha256:  sha256.New(),
		sha512:  sha512.New(),
		entropy: entropy.NewWriter(),
		ftype:   filetype.NewFiletypeWriter(),
		dst:     dst,
		size:    int64(0),
		node:    node,
		fs:      fs,
	}
	toReturn.mlt = io.MultiWriter(toReturn.md5, toReturn.sha1, toReturn.sha256, toReturn.sha512, toReturn.entropy, toReturn.ftype, dst)
	return toReturn
}

func (mwc *myWriteCloser) Write(p []byte) (int, error) {
	n, err := mwc.mlt.Write(p)
	mwc.size += int64(n)
	return n, err
}
func (mwc *myWriteCloser) Close() error {
	err := mwc.dst.Close()
	if err != nil {
		return err
	}
	ref := mwc.node.ref
	ref.typ = mwc.ftype.String()
	ref.md5 = hex.EncodeToString(mwc.md5.Sum(nil))
	ref.sha1 = hex.EncodeToString(mwc.sha1.Sum(nil))
	ref.sha256 = hex.EncodeToString(mwc.sha256.Sum(nil))
	ref.sha512 = hex.EncodeToString(mwc.sha512.Sum(nil))
	ref.entropy = mwc.entropy.Entropy()
	ref.size = mwc.size

	// TODO: think about using sync/atomic
	mwc.node.ref = mwc.fs.db.setIfEmpty(ref)
	if ref.id != mwc.node.ref.id {
		ref.remove(mwc.fs.storageDir)
	}

	return nil
}

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

	"github.com/jonathongardner/virtualfs/entropy"
	"github.com/jonathongardner/virtualfs/filetype"
)

var FileCachLimit = 1 * 1024 * 1024 // 1 MB
type CachedFile struct {
	cachedData []byte
	node       *FileInfo
	file       io.WriteCloser
}

func (mf *CachedFile) Write(p []byte) (int, error) {
	if mf.file != nil {
		return mf.file.Write(p)
	}
	mf.cachedData = append(mf.cachedData, p...)
	if len(mf.cachedData) >= FileCachLimit {
		return len(p), mf.saveFile()
	}
	return len(p), nil
}
func (mf *CachedFile) saveFile() error {
	dst, err := mf.node.ref.create(mf.node.db.storageDir)
	if err != nil {
		return fmt.Errorf("couldnt create cached file %v", err)
	}

	_, err = dst.Write(mf.cachedData)
	if err != nil {
		return fmt.Errorf("couldnt save cached file %v", err)
	}

	mf.file = dst
	mf.cachedData = []byte{}
	return nil
}

func (mf *CachedFile) CloseWithRef(ref *reference) error {
	// TODO: think about using sync/atomic
	mf.node.ref = mf.node.db.setIfEmpty(ref)
	if ref.id == mf.node.ref.id {
		// think about not saving empty file, though since we share
		// refences, it will only happen once and updating other code
		// to handle that might be work then worth...
		if mf.file == nil {
			return mf.saveFile()
		}
	} else {
		if mf.file != nil {
			return ref.remove(mf.node.db.storageDir)
		}
	}
	return nil
}

// Creates a file if not empty and cachs saving to `Close`
type myFile struct {
	md5     hash.Hash
	sha1    hash.Hash
	sha256  hash.Hash
	sha512  hash.Hash
	entropy *entropy.Writer
	size    int64
	ftype   *filetype.FiletypeWriter
	dst     *CachedFile
	mlt     io.Writer
}

func createMyWriterCloser(node *FileInfo) (*myFile, error) {
	dst := &CachedFile{make([]byte, 0), node, nil}
	toReturn := &myFile{
		md5:     md5.New(),
		sha1:    sha1.New(),
		sha256:  sha256.New(),
		sha512:  sha512.New(),
		entropy: entropy.NewWriter(),
		ftype:   filetype.NewFiletypeWriter(),
		dst:     dst,
		size:    int64(0),
	}
	toReturn.mlt = io.MultiWriter(toReturn.md5, toReturn.sha1, toReturn.sha256, toReturn.sha512, toReturn.entropy, toReturn.ftype, dst)
	return toReturn, nil
}

func (mwc *myFile) Write(p []byte) (int, error) {
	n, err := mwc.mlt.Write(p)
	mwc.size += int64(n)
	return n, err
}
func (mwc *myFile) Close() error {
	ref := mwc.dst.node.ref
	ref.typ = mwc.ftype.String()
	ref.md5 = hex.EncodeToString(mwc.md5.Sum(nil))
	ref.sha1 = hex.EncodeToString(mwc.sha1.Sum(nil))
	ref.sha256 = hex.EncodeToString(mwc.sha256.Sum(nil))
	ref.sha512 = hex.EncodeToString(mwc.sha512.Sum(nil))
	ref.entropy = mwc.entropy.Entropy()
	ref.size = mwc.size

	return mwc.dst.CloseWithRef(ref)
}

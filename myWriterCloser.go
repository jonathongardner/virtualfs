package virtualfs

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"

	"github.com/jonathongardner/virtualfs/entropy"
	"github.com/jonathongardner/virtualfs/filetype"
)

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
	node    *FileInfo
}

func createMyWriterCloser(node *FileInfo, dst io.WriteCloser) *myWriteCloser {
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
	mwc.node.ref = mwc.node.db.setIfEmpty(ref)
	if ref.id != mwc.node.ref.id {
		ref.remove(mwc.node.db.storageDir)
	}

	return nil
}

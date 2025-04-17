package virtualfs

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"

	"github.com/jonathongardner/fifo/entropy"
	"github.com/jonathongardner/virtualfs/filetype"
)

// ------------- multiwriter ------------------
type multiWriter struct {
	md5     hash.Hash
	sha1    hash.Hash
	sha256  hash.Hash
	sha512  hash.Hash
	entropy *entropy.Writer
	ftype   *filetype.FiletypeWriter
	size    int64
	mw      io.Writer
}

func newMultiWriter(w io.Writer) *multiWriter {
	toReturn := &multiWriter{
		md5:     md5.New(),
		sha1:    sha1.New(),
		sha256:  sha256.New(),
		sha512:  sha512.New(),
		entropy: entropy.NewWriter(),
		ftype:   filetype.NewFiletypeWriter(),
		size:    0,
	}
	toReturn.mw = io.MultiWriter(
		toReturn.md5,
		toReturn.sha1,
		toReturn.sha256,
		toReturn.sha512,
		toReturn.entropy,
		toReturn.ftype,
		w,
	)
	return toReturn
}

func (mw *multiWriter) Write(p []byte) (int, error) {
	n, err := mw.mw.Write(p)
	mw.size += int64(n)
	return n, err
}

func (mw *multiWriter) Value(value string) any {
	switch value {
	case "md5":
		return hex.EncodeToString(mw.md5.Sum(nil))
	case "sha1":
		return hex.EncodeToString(mw.sha1.Sum(nil))
	case "sha256":
		return hex.EncodeToString(mw.sha256.Sum(nil))
	case "sha512":
		return hex.EncodeToString(mw.sha512.Sum(nil))
	case "entropy":
		return mw.entropy.Entropy()
	case "filetype":
		return mw.ftype.Type()
	case "size":
		return mw.size
	default:
		panic("unknown hash type")
	}
}

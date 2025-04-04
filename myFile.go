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

	"github.com/jonathongardner/virtualfs/entropy"
	"github.com/jonathongardner/virtualfs/filetype"
)

type destination interface {
	Write(p []byte) (int, error)
	Keep() error
	Discard() error
}

// ------------- cachedFile ------------------
var FileCachLimit = 1 * 1024 * 1024 // 1 MB
// cachedFile is a file that caches the data until it is closed or the cache limit is reached
// it is used to avoid writing to disk too often for small files that have been seen alot
type cachedFile struct {
	cachedData     []byte
	path           string
	file           io.WriteCloser
	fileCacheLimit int
}

func defaultCacheFile(path string) *cachedFile {
	return &cachedFile{make([]byte, 0), path, nil, FileCachLimit}
}

// func noCacheFile(path string) *cachedFile {
// 	return &cachedFile{make([]byte, 0), path, nil, 0}
// }

// Write writes to the file if it exists, otherwise it caches the data
// until the file is closed or the cache limit is reached
func (mf *cachedFile) Write(p []byte) (int, error) {
	if mf.file != nil {
		return mf.file.Write(p)
	}
	mf.cachedData = append(mf.cachedData, p...)
	if len(mf.cachedData) >= mf.fileCacheLimit {
		return len(p), mf.saveFile()
	}
	return len(p), nil
}

// Keep saves the file to disk (if it didnt exist) and closes the file
func (mf *cachedFile) Keep() error {
	if mf.file == nil {
		if err := mf.saveFile(); err != nil {
			return err
		}
	}
	return mf.file.Close()
}

// Discard closes the file and removes it from the disk if it was created
func (mf *cachedFile) Discard() error {
	if mf.file != nil {
		if err := mf.file.Close(); err != nil {
			return err
		}
		return os.Remove(mf.path)
	}
	return nil
}

// saveFile saves the cached data to the file and empties the cache
func (mf *cachedFile) saveFile() error {
	dst, err := os.Create(mf.path)
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

//------------- cachedFile ------------------
//------------- mvFile ------------------
// mvFile is a file that writes to no where and moves the data to the destination if it is kept

type mvFile struct {
	newPath      string
	originalPath string
}

func defaultMvFile(path, originalPath string) *mvFile {
	return &mvFile{path, originalPath}
}

// Write writes to no where b/c the file will be moved at the end if kept
func (mf *mvFile) Write(p []byte) (int, error) {
	return len(p), nil
}

// Keep saves the file to disk (if it didnt exist) and closes the file
func (mf *mvFile) Keep() error {
	return os.Rename(mf.originalPath, mf.newPath)
}

// Discard closes the file and removes it from the disk if it was created
func (mf *mvFile) Discard() error {
	return nil
}

// ------------- myFile ------------------
// myFile an io.WriteCloser that calculates the md5, sha1, sha256, sha512, entropy and filetype of the file
// and saves it to disk
type myFile struct {
	md5     hash.Hash
	sha1    hash.Hash
	sha256  hash.Hash
	sha512  hash.Hash
	entropy *entropy.Writer
	size    int64
	ftype   *filetype.FiletypeWriter
	dst     destination
	mlt     io.Writer
	node    *FileInfo
}

// createCachedMyWriterCloser creates a new myFile writer with cached file
func createMyWriterCloser(node *FileInfo, dst destination) (*myFile, error) {
	toReturn := &myFile{
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
	return toReturn, nil
}

// Write write the date to all the writers and calculates the size
func (mwc *myFile) Write(p []byte) (int, error) {
	n, err := mwc.mlt.Write(p)
	mwc.size += int64(n)
	return n, err
}

// Close set the type, hashes, etc and closes keeps or discards the file dpending on if it is a duplicate
func (mwc *myFile) Close() error {
	ref := mwc.node.ref
	ref.typ = mwc.ftype.String()
	ref.md5 = hex.EncodeToString(mwc.md5.Sum(nil))
	ref.sha1 = hex.EncodeToString(mwc.sha1.Sum(nil))
	ref.sha256 = hex.EncodeToString(mwc.sha256.Sum(nil))
	ref.sha512 = hex.EncodeToString(mwc.sha512.Sum(nil))
	ref.entropy = mwc.entropy.Entropy()
	ref.size = mwc.size

	// if updated than we have seen this sha512 before so no need to keep the file
	if mwc.node.updateIfDuplicateRef() {
		return mwc.dst.Discard()
	}
	return mwc.dst.Keep()
}

//------------- myFile ------------------

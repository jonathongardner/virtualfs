package virtualfs

import (
	"fmt"

	"github.com/jonathongardner/fifo/buffer"
	"github.com/jonathongardner/fifo/identifiers"
)

type destination interface {
	Write(p []byte) (int, error)
	Close() error
	Delete() error
}

var bufferSize = 512 * 1024 * 1024 // 512 * 1MB

// ------------- myFile ------------------
// myFile an io.WriteCloser that calculates the md5, sha1, sha256, sha512, entropy and filetype of the file
// and saves it to disk
type myFile struct {
	identifiers *identifiers.Writer
	file        destination
	node        *Fs
}

// createCachedMyWriterCloser creates a new myFile writer with cached file
func createMyWriterCloser(node *Fs, path string) (*myFile, error) {
	file, err := buffer.NewFileWriter(path, bufferSize)
	if err != nil {
		return nil, err
	}
	toReturn := &myFile{
		identifiers: identifiers.NewWriter(file),
		file:        file,
		node:        node,
	}
	return toReturn, nil
}

func (mwc *myFile) Write(p []byte) (int, error) {
	return mwc.identifiers.Write(p)
}

// Close set the type, hashes, etc and closes keeps or discards the file dpending on if it is a duplicate
func (mwc *myFile) Close() error {
	if err := mwc.identifiers.Close(); err != nil {
		return fmt.Errorf("error closing file %w", err)
	}

	identifiers, err := mwc.identifiers.Identifiers()
	if err != nil {
		return fmt.Errorf("error getting identifiers %w", err)
	}

	ref := mwc.node.ref
	ref.size = identifiers.Size
	ref.md5 = identifiers.Md5
	ref.sha1 = identifiers.Sha1
	ref.sha256 = identifiers.Sha256
	ref.sha512 = identifiers.Sha512
	ref.entropy = identifiers.Entropy
	ref.typ = identifiers.Filetype

	// if updated than we have seen this sha512 before so no need to keep the file
	if mwc.node.updateIfDuplicateRef() {
		if err := mwc.file.Delete(); err != nil {
			return fmt.Errorf("error deleting file %w", err)
		}
	}
	return mwc.file.Close()
}

//------------- myFile ------------------

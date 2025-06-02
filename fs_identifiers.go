package virtualfs

import (
	"fmt"

	"github.com/jonathongardner/fifo/filetype"
)

// ID returns the id of the file
func (n *Fs) ID() string {
	return n.ref.id
}

// Size is needed for os.FileInfo interface

// Filetype returns the filetype of the file
func (fi *Fs) Filetype() filetype.Filetype {
	return fi.ref.typ
}

// Mimetype returns the mimetype of the file
func (fi *Fs) Mimetype() string {
	return fi.ref.typ.Mimetype
}

// sha1 returns the sha1 of the file
func (fi *Fs) Md5() string {
	return fi.ref.md5
}

// sha1 returns the sha1 of the file
func (fi *Fs) Sha1() string {
	return fi.ref.sha1
}

// sha1 returns the sha1 of the file
func (fi *Fs) Sha256() string {
	return fi.ref.sha256
}

// Sha512 returns the sha512 of the file
func (fi *Fs) Sha512() string {
	return fi.ref.sha512
}

// sha1 returns the sha1 of the file
func (fi *Fs) Entropy() float64 {
	return fi.ref.entropy
}

// ErrorId returns the error id of the file
func (fi *Fs) ErrorId() error {
	return fmt.Errorf("id: %v, name: %v, type: %v,", fi.ref.id, fi.name, fi.ref.typ)
}

// func (fi *Fs) specialType() bool {
// 	return fi.ref.typ == filetype.Directory || fi.ref.typ == filetype.Symlink
// }

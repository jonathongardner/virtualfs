package virtualfs

import (
	"os"
	"time"
)

// ---------------------FileInfo Methods--------------------
func (fi *Fs) Name() string {
	return fi.name
}
func (fi *Fs) Size() int64 {
	return fi.ref.size
}
func (fi *Fs) Mode() os.FileMode {
	return fi.mode
}
func (fi *Fs) ModTime() time.Time {
	return fi.modTime
}
func (fi *Fs) IsDir() bool {
	return fi.mode.IsDir()
}
func (fi *Fs) Sys() any {
	return nil
}

// ---------------------FileInfo Methods--------------------

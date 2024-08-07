package virtualfs

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	// log "github.com/sirupsen/logrus"
)

type Fs struct {
	root   *FileInfo
	closed bool
}

// var root = &Entry{type: filetype.Directory}
var ErrDontWalk = fmt.Errorf("dont walk entries children")
var ErrClosed = fmt.Errorf("virtual file system is closed")
var ErrNotFound = fmt.Errorf("file not found") // https://smyrman.medium.com/writing-constant-errors-with-go-1-13-10c4191617
var ErrOutsideFilesystem = fmt.Errorf("path is outside of filesystem")

func NewFs(storageDir, rootPath string) (*Fs, error) {
	err := os.Mkdir(storageDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to create storage dir: %v", err)
	}

	reader := os.Stdin
	mode := os.FileMode(0755)
	modTime := time.Now()
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
		modTime = fileInfo.ModTime()
		reader = fileToCopy
	}
	toReturn := &Fs{root: newFileInfo(newReferenceDB(storageDir), filename, mode, modTime)}

	writer, err := toReturn.root.create()
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		return nil, fmt.Errorf("couldn't copy file (%v) - %v", rootPath, err)
	}

	return toReturn, nil
}

func NewFsFromDir(storageDir string) (*Fs, error) {
	toReturn := &Fs{}
	return toReturn, toReturn.load(storageDir)
}

func (v *Fs) Close() error {
	v.closed = true
	return v.save()
}

func (v *Fs) checkClosed() error {
	if v.closed {
		return ErrClosed
	}
	return nil
}

func (v *Fs) FsFrom(path string) (*Fs, error) {
	err := v.checkClosed()
	if err != nil {
		return nil, err
	}

	toWalk, _, err := v.fileInfoFrom(path, -1)
	if err != nil {
		return nil, fmt.Errorf("%v: %v (FsFrom)", err, path)
	}

	return &Fs{root: toWalk}, nil
}

// func (v *Fs) FsChildren() (toReturn []*Fs) {
// 	if v.root.ref.children == nil {
// 		return
// 	}

// 	for _, n := range v.root.ref.children {
// 		toReturn = append(toReturn, &Fs{root: n})
// 	}
// 	return
// }

// --------Root stuff----------
func (v *Fs) Process() error {
	return v.root.ref.process()
}

//--------Root stuff----------

// ---------------------Disk Operations--------------------
func (v *Fs) Stat(path string) (*FileInfo, error) {
	err := v.checkClosed()
	if err != nil {
		return nil, err
	}

	toWalk, _, err := v.fileInfoFrom(path, -1)
	if err != nil {
		return nil, err
	}
	return toWalk, nil
}
func (v *Fs) StatAt(path string, at int) (*FileInfo, error) {
	err := v.checkClosed()
	if err != nil {
		return nil, err
	}

	toWalk, _, err := v.fileInfoFrom(path, at)
	if err != nil {
		return nil, err
	}
	return toWalk, nil
}

func (v *Fs) Open(path string) (*os.File, error) {
	err := v.checkClosed()
	if err != nil {
		return nil, err
	}

	toWalk, _, err := v.fileInfoFrom(path, -1)
	if err != nil {
		return nil, err
	}
	return toWalk.Open()
}

func (v *Fs) MkdirP(path string, perm os.FileMode, modTime time.Time) error {
	err := v.checkClosed()
	if err != nil {
		return err
	}

	paths, err := split(path)
	if err != nil {
		return err
	}
	_, err = v.root.mkdirP(paths, perm, modTime)
	return err
}
func (v *Fs) CreateChild(perm os.FileMode, modTime time.Time) (*myWriteCloser, error) {
	err := v.checkClosed()
	if err != nil {
		return nil, err
	}

	newFileInfo, err := v.root.touch([]string{}, perm, modTime)
	if err != nil {
		return nil, err
	}

	return newFileInfo.create()
}
func (v *Fs) Create(path string, perm os.FileMode, modTime time.Time) (*myWriteCloser, error) {
	err := v.checkClosed()
	if err != nil {
		return nil, err
	}

	paths, err := split(path)
	if err != nil {
		return nil, err
	}
	if len(path) == 0 {
		panic("path shoudnt be empty")
	}

	newFileInfo, err := v.root.touch(paths, perm, modTime)
	if err != nil {
		return nil, err
	}

	return newFileInfo.create()
}

func (v *Fs) Symlink(oldname, newname string, perm os.FileMode, modTime time.Time) error {
	err := v.checkClosed()
	if err != nil {
		return err
	}

	paths, err := split(newname)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		panic("path shoudnt be empty")
	}

	_, err = v.root.symlink(oldname, paths, perm, modTime)
	return err
}

// ---------------------Disk Operations--------------------

func (v *Fs) Walk(path string, callback func(string, *FileInfo) error) error {
	toWalk, path, err := v.fileInfoFrom(path, -1)
	if err != nil {
		return fmt.Errorf("%v: %v (walk)", err, path)
	}
	return toWalk.walkRecursive(path, callback)
}

// -1 returns that last one
func (v *Fs) fileInfoFrom(path string, at int) (*FileInfo, string, error) {
	paths, err := split(path)
	if err != nil {
		return nil, "", err
	}
	// clean path
	path = filepath.Join("/", filepath.Join(paths...))

	toReturn, err := v.root.travelTo(paths, at)
	return toReturn, path, err
}

// ------------------split------------------
func split(dir string) ([]string, error) {
	// vulnerable to path traversal
	// dir := filepath.Clean(toClean)

	toReturn := []string{}
	for _, p := range strings.Split(dir, "/") {
		if p == "" || p == "." {
			continue
		} else if p == ".." {
			if len(toReturn) == 0 {
				return nil, ErrOutsideFilesystem
			}
			toReturn = toReturn[:len(toReturn)-1]
		} else {
			toReturn = append(toReturn, p)
		}
	}
	return toReturn, nil
}

//------------------split------------------

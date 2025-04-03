package virtualfs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"time"
	// log "github.com/sirupsen/logrus"
)

type Fs struct {
	root   *FileInfo
	parent *Fs
	closed bool
}

// var root = &Entry{type: filetype.Directory}
var ErrDontWalk = fmt.Errorf("dont walk entries children")
var ErrNotFound = fmt.Errorf("file not found") // https://smyrman.medium.com/writing-constant-errors-with-go-1-13-10c4191617
var ErrOutsideFilesystem = fmt.Errorf("path is outside of filesystem")
var ErrInFilesystem = fmt.Errorf("filesystem errors")

// NewFsFromDb loads a virtual file system from a directory DB
func NewFsFromDb(storageDir string) (*Fs, error) {
	toReturn := &Fs{}
	return toReturn, toReturn.load(storageDir)
}

// NewFs creates a new virtual file system from a file or stdin
func NewFs(storageDir, rootPath string) (*Fs, error) {
	err := os.Mkdir(storageDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to create storage dir: %v", err)
	}

	if rootPath == "" {
		return newFileFs(storageDir, "stdin", 0755, time.Now(), os.Stdin)
	}

	fileToCopy, err := os.Open(rootPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't open path (%v) - %v", rootPath, err)
	}
	defer fileToCopy.Close()

	fileInfo, err := fileToCopy.Stat()
	if err != nil {
		return nil, fmt.Errorf("couldn't get path info (%v) - %v", rootPath, err)
	}
	if !fileInfo.IsDir() {
		return newFileFs(storageDir, path.Base(rootPath), fileInfo.Mode(), fileInfo.ModTime(), fileToCopy)
	}

	return newFileDirFs(storageDir, path.Base(rootPath), fileInfo.Mode(), fileInfo.ModTime(), rootPath)
}

// newFileFs creates the the virtual file system copies the file to the virtual system
func newFileFs(storageDir, filename string, mode os.FileMode, modTime time.Time, reader io.Reader) (*Fs, error) {
	toReturn := &Fs{root: newFileInfo(newReferenceDB(storageDir), filename, mode, modTime)}

	writer, err := toReturn.root.create()
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	if err != nil {
		return nil, fmt.Errorf("couldn't copy file (%v) - %v", filename, err)
	}

	return toReturn, nil
}

// newFileDirFs creates the virtual file system from the dir
func newFileDirFs(storageDir, filename string, mode os.FileMode, modTime time.Time, dir string) (*Fs, error) {
	toReturn := &Fs{root: newDirFileInfo(newReferenceDB(storageDir), filename, mode, modTime)}

	wg := newErrorWG()
	wg.run(func() error { return toReturn.addDirFiles(dir, wg) })
	err := wg.wait()
	if err != nil {
		return nil, err
	}

	return toReturn, nil
}

// addDirFiles adds all files in the directory to the virtual file system
func (fs *Fs) addDirFiles(dirPath string, wg *errorWG) error {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("couldn't read dir (%v) - %v", dirPath, err)
	}

	for _, dirEntry := range dirEntries {
		wg.run(func() error { return fs.addEntry(dirPath, dirEntry, wg) })
	}
	return nil
}

// addEntry adds dir entry to virutal fs and recursively adds subdirectories
func (fs *Fs) addEntry(dirPath string, dirEntry fs.DirEntry, wg *errorWG) error {
	name := dirEntry.Name()
	fullPath := filepath.Join(dirPath, name)
	fileInfo, err := dirEntry.Info()
	if err != nil {
		return fmt.Errorf("couldn't get file info (%v) - %v", fullPath, err)
	}

	if dirEntry.IsDir() {
		// Recursively add subdirectories
		err := fs.MkdirP(name, fileInfo.Mode(), fileInfo.ModTime())
		if err != nil {
			return err
		}
		newFs, err := fs.FsFrom(name)
		if err != nil {
			return err
		}

		wg.run(func() error { return newFs.addDirFiles(fullPath, wg) })
		return nil
	}

	// Create a new file in the virtual filesystem
	newFile, err := fs.Create(name, fileInfo.Mode(), fileInfo.ModTime())
	if err != nil {
		return err
	}
	defer newFile.Close()

	// Copy the contents of the file to the virtual filesystem
	srcFile, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("couldn't open file (%v) - %v", fullPath, err)
	}
	defer srcFile.Close()

	_, err = io.Copy(newFile, srcFile)
	if err != nil {
		return fmt.Errorf("couldn't copy file (%v) - %v", fullPath, err)
	}

	return nil
}

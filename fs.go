package virtualfs

import (
	"fmt"
	"io"
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
func NewFs(storageDir, rootPath string, move bool) (*Fs, error) {
	err := os.Mkdir(storageDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("unable to create storage dir: %v", err)
	}

	if rootPath == "" {
		if move {
			return nil, fmt.Errorf("stdin not supported with move")
		}
		fs := smartNewFs(storageDir, "stdin", 0755, time.Now())
		if err := fs.copyReader(os.Stdin); err != nil {
			return nil, fmt.Errorf("couldn't copy stdin - %v", err)
		}
		return fs, nil
	}

	fileInfo, err := os.Stat(rootPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't get path info (%v) - %v", rootPath, err)
	}

	fs := smartNewFs(storageDir, path.Base(rootPath), fileInfo.Mode(), fileInfo.ModTime())

	wg := newErrorWG()
	wg.run(func() error { return fs.addEntry(rootPath, move, wg) })
	err = wg.wait()
	if err != nil {
		return nil, err
	}

	return fs, nil
}

func smartNewFs(storageDir, filename string, mode os.FileMode, modTime time.Time) *Fs {
	toReturn := &Fs{root: newFileInfo(newReferenceDB(storageDir), filename, mode, modTime)}
	if mode.IsDir() {
		toReturn.root.setToDir()
	}
	return toReturn
}

// addDirFiles adds all files in the directory to the virtual file system
func (fs *Fs) addDirFiles(dirPath string, move bool, wg *errorWG) error {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("couldn't read dir (%v) - %v", dirPath, err)
	}

	for _, dirEntry := range dirEntries {
		wg.run(func() error {
			name := dirEntry.Name()
			fileInfo, err := dirEntry.Info()
			if err != nil {
				return fmt.Errorf("couldn't get file info (%v/%v) - %v", dirPath, name, err)
			}

			newFs, err := fs.smartNewFs(name, fileInfo.Mode(), fileInfo.ModTime())
			if err != nil {
				return err
			}
			return newFs.addEntry(filepath.Join(dirPath, name), move, wg)
		})
	}
	return nil
}

// addEntry adds dir entry to virutal fs and recursively adds subdirectories
func (fs *Fs) addEntry(path string, move bool, wg *errorWG) error {
	if fs.root.IsDir() {
		return fs.addDirFiles(path, move, wg)
	}

	// Copy the contents of the file to the virtual filesystem
	srcFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("couldn't open file (%v) - %v", path, err)
	}
	defer srcFile.Close()

	if move {
		err = fs.moveReader(path, srcFile)
	} else {
		err = fs.copyReader(srcFile)
	}
	if err != nil {
		return fmt.Errorf("couldn't copy file (%v) - %v", path, err)
	}

	return nil
}

// copyReader copies the contents of a reader to the virtual filesystem
func (fs *Fs) moveReader(path string, src io.Reader) error {
	newFile, err := fs.root.createMv(path)
	if err != nil {
		return err
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, src)
	if err != nil {
		return err
	}

	return nil
}

// copyReader copies the contents of a reader to the virtual filesystem
func (fs *Fs) copyReader(src io.Reader) error {
	newFile, err := fs.root.create()
	if err != nil {
		return err
	}
	defer newFile.Close()

	_, err = io.Copy(newFile, src)
	if err != nil {
		return err
	}

	return nil
}

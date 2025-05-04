package virtualfs

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"
	// log "github.com/sirupsen/logrus"
)

type Fs struct {
	root   *FileInfo
	parent *Fs
	closed bool
}

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
	if !fileInfo.IsDir() {
		file, err := os.Open(rootPath)
		if err != nil {
			return nil, fmt.Errorf("couldn't open file (%v) - %v", rootPath, err)
		}
		defer file.Close()

		if err := fs.copyReader(file); err != nil {
			return nil, fmt.Errorf("couldn't copy file (%v) - %v", rootPath, err)
		}
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

// copyReader copies the contents of a reader to the virtual filesystem
func (fs *Fs) copyReader(src io.Reader) error {
	newFile, err := fs.root.Create()
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

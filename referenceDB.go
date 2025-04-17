package virtualfs

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/jonathongardner/fifo/bounded"
)

var maxReadFiles = 10

type referenceDB struct {
	wf     *os.File          // write file
	bfp    *bounded.FilePool // read file pool
	mapMu  sync.Mutex
	wfMu   sync.Mutex
	err    bool
	warn   bool
	refMap map[string]*reference
}

func newReferenceDB(wf *os.File) *referenceDB {
	// only returns error if size out of bounds
	bfp, _ := bounded.NewFilePool(wf.Name(), maxReadFiles)
	return &referenceDB{wf: wf, bfp: bfp, err: false, warn: false, refMap: make(map[string]*reference)}
}

// setIfEmpty checks if the reference already exists in the map
// if it does, it returns the existing reference and false
// if it doesn't, it adds the reference to the map and returns true
func (rdb *referenceDB) setIfEmpty(passedRef *reference) (*reference, bool) {
	rdb.mapMu.Lock()
	defer rdb.mapMu.Unlock()

	if ref, ok := rdb.refMap[passedRef.sha512]; ok {
		return ref, false
	}

	rdb.refMap[passedRef.sha512] = passedRef
	return passedRef, true
}

func (rdb *referenceDB) matchSet(passedRef *reference) bool {
	rdb.mapMu.Lock()
	defer rdb.mapMu.Unlock()

	if ref, ok := rdb.refMap[passedRef.sha512]; ok {
		return ref.id == passedRef.id
	}

	return false
}

func (rdb *referenceDB) lockWriteFile() {
	rdb.mapMu.Lock()
}

func (rdb *referenceDB) unlockWriteFile() {
	rdb.mapMu.Unlock()
}

func (rdb *referenceDB) add(passedRef *reference, callback func(file *os.File) (int64, error)) error {
	if passedRef.offset != -1 {
		return fmt.Errorf("reference already has offset")
	}

	rdb.wfMu.Lock()
	defer rdb.wfMu.Unlock()

	var err error
	passedRef.offset, err = rdb.wf.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("error getting offset for db - %v", err)
	}

	passedRef.compressedSize, err = callback(rdb.wf)
	if err != nil {
		return err
	}

	return nil
}

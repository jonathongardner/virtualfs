package virtualfs

import (
	"path/filepath"
	"sync"
)

type referenceDB struct {
	storageDir string
	mu         sync.Mutex
	err        bool
	warn       bool
	refMap     map[string]*reference
}

func newReferenceDB(storageDir string) *referenceDB {
	return &referenceDB{storageDir: storageDir, err: false, warn: false, refMap: make(map[string]*reference)}
}

func (rdb *referenceDB) setIfEmpty(passedRef *reference) (*reference, bool) {
	rdb.mu.Lock()
	defer rdb.mu.Unlock()

	if ref, ok := rdb.refMap[passedRef.sha512]; ok {
		return ref, false
	}

	rdb.refMap[passedRef.sha512] = passedRef
	return passedRef, true
}

func (rdb *referenceDB) finDBPath() string {
	return filepath.Join(rdb.storageDir, "fin.db")
}

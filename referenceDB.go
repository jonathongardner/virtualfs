package virtualfs

import (
	"sync"
)

type referenceDB struct {
	storageDir string
	mu         sync.Mutex
	refMap     map[string]*reference
}

func newReferenceDB(storageDir string) *referenceDB {
	return &referenceDB{storageDir: storageDir, refMap: make(map[string]*reference)}
}

func (rdb *referenceDB) setIfEmpty(passedRef *reference) *reference {
	rdb.mu.Lock()
	defer rdb.mu.Unlock()

	if ref, ok := rdb.refMap[passedRef.sha512]; ok {
		return ref
	}

	rdb.refMap[passedRef.sha512] = passedRef
	return passedRef
}

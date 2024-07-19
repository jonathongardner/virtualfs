package virtualfs

import (
	"sync"
)

type referenceDB struct {
	mu     sync.Mutex
	refMap map[string]*reference
}

func newReferenceDB() *referenceDB {
	return &referenceDB{refMap: make(map[string]*reference)}
}

func (rdb *referenceDB) setIfEmpty(ref *reference) *reference {
	rdb.mu.Lock()
	defer rdb.mu.Unlock()

	if ref, ok := rdb.refMap[ref.sha512]; ok {
		return ref
	}

	rdb.refMap[ref.sha512] = ref
	return ref
}

package virtualfs

// ------------- cachedFile ------------------
// var FileCachLimit = 1 * 1024 * 1024 // 1 MB
// // cachedFile is a file that caches the data until it is closed or the cache limit is reached
// // it is used to avoid writing to disk too often for small files that have been seen alot
// type cachedFile struct {
// 	cachedData     []byte
// 	path           string
// 	file           io.WriteCloser
// 	fileCacheLimit int
// }

// func defaultCacheFile(path string) *cachedFile {
// 	return &cachedFile{make([]byte, 0), path, nil, FileCachLimit}
// }

// func noCacheFile(path string) *cachedFile {
// 	return &cachedFile{make([]byte, 0), path, nil, 0}
// }

// Write writes to the file if it exists, otherwise it caches the data
// until the file is closed or the cache limit is reached
// func (mf *cachedFile) Write(p []byte) (int, error) {
// 	if mf.file != nil {
// 		return mf.file.Write(p)
// 	}
// 	mf.cachedData = append(mf.cachedData, p...)
// 	if len(mf.cachedData) >= mf.fileCacheLimit {
// 		return len(p), mf.saveFile()
// 	}
// 	return len(p), nil
// }

// // Keep saves the file to disk (if it didnt exist) and closes the file
// func (mf *cachedFile) Keep() error {
// 	if mf.file == nil {
// 		if err := mf.saveFile(); err != nil {
// 			return err
// 		}
// 	}
// 	return mf.file.Close()
// }

// // Discard closes the file and removes it from the disk if it was created
// func (mf *cachedFile) Discard() error {
// 	if mf.file != nil {
// 		if err := mf.file.Close(); err != nil {
// 			return err
// 		}
// 		return os.Remove(mf.path)
// 	}
// 	return nil
// }

// // saveFile saves the cached data to the file and empties the cache
// func (mf *cachedFile) saveFile() error {
// 	dst, err := os.Create(mf.path)
// 	if err != nil {
// 		return fmt.Errorf("couldnt create cached file %v", err)
// 	}

// 	_, err = dst.Write(mf.cachedData)
// 	if err != nil {
// 		return fmt.Errorf("couldnt save cached file %v", err)
// 	}

// 	mf.file = dst
// 	mf.cachedData = []byte{}
// 	return nil
// }

//------------- cachedFile ------------------

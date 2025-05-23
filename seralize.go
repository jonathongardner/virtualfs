package virtualfs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jonathongardner/fifo/filetype"
)

func (v *Fs) FinDBPath() string {
	return v.root.db.finDBPath()
}

func (v *Fs) DBDir() string {
	return v.root.db.storageDir
}

func (v *Fs) save() error {
	dbFile := v.FinDBPath()
	file, err := os.Create(dbFile)
	if err != nil {
		return fmt.Errorf("error opneing file %v - %v", dbFile, err)
	}
	defer file.Close()

	err = v.Walk("/", func(path string, info *FileInfo) error {
		toSave := map[string]any{"path": path, "info": fileInfoToMap(info)}
		jsonString, _ := json.Marshal(toSave)
		// encoder := json.NewEncoder(file)
		// encoder.Encode(toSave)
		_, err := file.Write(jsonString)
		if err != nil {
			return err
		}
		_, err = file.WriteString("\n")
		return err
	})
	if err != nil {
		return err
	}

	return nil
}

func (v *Fs) load(storageDir string) error {
	db := newReferenceDB(storageDir)

	dbFile := db.finDBPath()
	file, err := os.Open(dbFile)
	if err != nil {
		return fmt.Errorf("error opening db file - %v", err)
	}
	defer file.Close()

	sc := bufio.NewScanner(file)
	for sc.Scan() {
		toLoad := make(map[string]any)
		err := json.Unmarshal(sc.Bytes(), &toLoad)
		if err != nil {
			return fmt.Errorf("error decoding db line - %v", err)
		}
		path := toLoad["path"].(string)

		fileInfo, err := mapToFileInfo(db, toLoad["info"].(map[string]any))
		if err != nil {
			return fmt.Errorf("error decoding FileInfo - %v", err)
		}

		if path == "/" {
			v.root = fileInfo
		} else {
			if v.root == nil {
				return fmt.Errorf("root FileInfo not set")
			}
			parentNode := v.root
			paths, err := split(path)
			if err != nil {
				return err
			}
			end := len(paths) - 1
			for _, p := range paths[:end] {
				var ok bool
				parentNode, ok = parentNode.ref.children[p]
				if !ok {
					return fmt.Errorf("error getting FileInfo %v", path)
				}
			}

			parentNode.ref.children[paths[end]] = fileInfo
		}
	}

	err = sc.Err()
	if err != nil {
		return fmt.Errorf("error reading db - %v", err)
	}

	err = os.Remove(dbFile)
	if err != nil {
		return fmt.Errorf("error removing db - %v", err)
	}

	if v.root == nil {
		return fmt.Errorf("no paths in database")
	}

	return nil
}

// ------------------------------JSON stuff--------------------------------
func fileInfoToMap(n *FileInfo) map[string]any {
	toReturn := make(map[string]any)
	toReturn["name"] = n.name
	toReturn["mode"] = uint32(n.mode)
	toReturn["modTime"] = n.modTime

	toReturn["id"] = n.ref.id
	toReturn["type"] = n.ref.typ
	tags := map[string]any{}
	n.ref.tags.Range(func(key, value any) bool {
		tags[key.(string)] = value
		return true // Return true to continue iterating
	})
	toReturn["tags"] = tags
	if n.ref.err != nil {
		toReturn["error"] = n.ref.err.Error()
	}
	if n.ref.warn != nil {
		warning := []string{}
		for _, e := range n.ref.warn {
			warning = append(warning, e.Error())
		}
		toReturn["warning"] = warning
	}
	if n.ref.typ == filetype.Symlink {
		toReturn["symlink"] = n.symlinkPath
	} else if n.ref.typ != filetype.Dir {
		toReturn["size"] = n.ref.size
		toReturn["md5"] = n.ref.md5
		toReturn["sha1"] = n.ref.sha1
		toReturn["sha256"] = n.ref.sha256
		toReturn["sha512"] = n.ref.sha512
		toReturn["entropy"] = n.ref.entropy
	}
	return toReturn
}

func mapToFileInfo(db *referenceDB, data map[string]any) (*FileInfo, error) {
	var err error
	var ok bool
	n := &FileInfo{db: db}
	n.name, ok = data["name"].(string)
	if !ok {
		return nil, fmt.Errorf("error getting name: %v", data["name"])
	}

	mod, ok := data["mode"].(float64)
	if !ok {
		return nil, fmt.Errorf("error getting mode: %v", data["mode"])
	}
	n.mode = os.FileMode(uint32(mod))

	modTime, ok := data["modTime"].(string)
	if !ok {
		return nil, fmt.Errorf("error getting modTime: %v", data["modTime"])
	}
	n.modTime, err = time.Parse(time.RFC3339, modTime)
	if err != nil {
		return nil, fmt.Errorf("error parsing modTime %v", err)
	}

	ref := &reference{children: make(map[string]*FileInfo)}
	n.ref = ref

	ref.id, ok = data["id"].(string)
	if !ok {
		return nil, fmt.Errorf("error getting id: %v", data["id"])
	}

	typ, ok := data["type"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("error getting type: %v", data["type"])
	}
	ref.typ = filetype.FiletypeFromJson(typ)

	tags, ok := data["tags"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("error getting tags: %v", data["tags"])
	}
	for k, v := range tags {
		ref.tags.Store(k, v)
	}

	_, ok = data["error"]
	if ok {
		errValue, ok := data["error"].(string)
		if !ok {
			return nil, fmt.Errorf("error getting error: %v", data["error"])
		}
		db.err = true
		ref.err = fmt.Errorf(errValue)
	}

	_, ok = data["warning"]
	if ok {
		warnValues, ok := data["warning"].([]any)
		if !ok {
			return nil, fmt.Errorf("error getting warning: %v", data["warning"])
		}
		db.warn = true
		for i, warnValue := range warnValues {
			wv, ok := warnValue.(string)
			if !ok {
				return nil, fmt.Errorf("error getting warning.%v: %v", i, wv)
			}
			ref.warn = append(ref.warn, fmt.Errorf(wv))
		}
	}

	if ref.typ == filetype.Dir {
		n.ref = ref
	} else if ref.typ == filetype.Symlink {
		n.symlinkPath, ok = data["symlink"].(string)
		if !ok {
			return nil, fmt.Errorf("error getting symlinkPath: %v", data["symlink"])
		}
		n.ref = ref
	} else {
		size, ok := data["size"].(float64)
		if !ok {
			return nil, fmt.Errorf("error getting size: %v", data["size"])
		}
		ref.size = int64(size)

		ref.md5, ok = data["md5"].(string)
		if !ok {
			return nil, fmt.Errorf("error getting md5: %v", data["md5"])
		}
		ref.sha1, ok = data["sha1"].(string)
		if !ok {
			return nil, fmt.Errorf("error getting sha1: %v", data["sha1"])
		}
		ref.sha256, ok = data["sha256"].(string)
		if !ok {
			return nil, fmt.Errorf("error getting sha256: %v", data["sha256"])
		}
		ref.sha512, ok = data["sha512"].(string)
		if !ok {
			return nil, fmt.Errorf("error getting sha512: %v", data["sha512"])
		}
		ref.entropy, ok = data["entropy"].(float64)
		if !ok {
			return nil, fmt.Errorf("error getting entropy: %v", data["entropy"])
		}

		// if its a file, therfore it has a checksum so use this
		path := filepath.Join(db.storageDir, ref.id)
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("error getting file %v", err)
		}
		// if sha512 already seen it will return that reference otherwise it creates a new one
		n.updateIfDuplicateRef()
	}

	return n, nil
}

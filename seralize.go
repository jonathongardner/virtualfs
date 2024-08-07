package virtualfs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/jonathongardner/virtualfs/filetype"
)

func (v *Fs) save() error {
	dbFile := filepath.Join(v.storageDir, "fin.db")
	file, err := os.Create(dbFile)
	if err != nil {
		return fmt.Errorf("error opneing file %v - %v", dbFile, err)
	}
	defer file.Close()

	count := 0
	err = v.Walk("/", func(path string, info os.FileInfo) error {
		toSave := map[string]any{"path": path, "info": info}
		if info.Mode().IsRegular() {
			count++
		}
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

func (v *Fs) load() error {
	dbFile := filepath.Join(v.storageDir, "fin.db")
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

		node, err := v.mapToNode(toLoad["info"].(map[string]any))
		if err != nil {
			return fmt.Errorf("error decoding node - %v", err)
		}

		if path == "/" {
			v.root = node
		} else {
			if v.root == nil {
				return fmt.Errorf("root node not set")
			}
			parentNode := v.root
			paths := split(path)
			end := len(paths) - 1
			for _, p := range paths[:end] {
				var ok bool
				parentNode, ok = parentNode.ref.children[p]
				if !ok {
					return fmt.Errorf("error getting node %v", path)
				}
			}

			parentNode.ref.children[paths[end]] = node
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

func (v *Fs) mapToNode(data map[string]any) (*node, error) {
	var err error
	var ok bool
	n := &node{}
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

	ref := &reference{processed: &atomic.Bool{}, children: make(map[string]*node)}

	ref.id, ok = data["id"].(string)
	if !ok {
		return nil, fmt.Errorf("error getting id: %v", data["id"])
	}

	typ, ok := data["type"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("error getting type: %v", data["type"])
	}
	ref.typ = filetype.FiletypeFromJson(typ)

	pr, ok := data["processed"].(bool)
	if !ok {
		return nil, fmt.Errorf("error getting processed: %v", data["processed"])
	}
	ref.processed.Store(pr)

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
		path := filepath.Join(v.storageDir, ref.id)
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("error getting file %v", err)
		}
		// if sha512 already seen it will return that reference otherwise it creates a new one
		n.ref = v.db.setIfEmpty(ref)
	}

	return n, nil
}

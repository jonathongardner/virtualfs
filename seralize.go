package virtualfs

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jonathongardner/fifo/filetype"
)

func (v *Fs) FinDBPath() string {
	return v.db.finDBPath()
}

func (v *Fs) DBDir() string {
	return v.db.storageDir
}

func (v *Fs) save() error {
	dbFile := v.FinDBPath()
	file, err := os.Create(dbFile)
	if err != nil {
		return fmt.Errorf("error opneing file %v - %w", dbFile, err)
	}
	defer file.Close()

	err = v.walkRecursive("/", false, func(path string, child bool, fs *Fs) error {
		jsonString, err := json.Marshal(toJsonFs(path, child, fs))
		if err != nil {
			return fmt.Errorf("error marshalling file info %v - %w", path, err)
		}
		// encoder := json.NewEncoder(file)
		// encoder.Encode(toSave)
		_, err = file.Write(jsonString)
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
	v.db = newReferenceDB(storageDir)

	dbFile := v.db.finDBPath()
	file, err := os.Open(dbFile)
	if err != nil {
		return fmt.Errorf("error opening db file - %w", err)
	}
	defer file.Close()

	sc := bufio.NewScanner(file)

	sc.Scan()
	jsonFs, err := unmarshal(sc.Bytes())
	if err != nil {
		return fmt.Errorf("error unmarshalling root Fs - %w", err)
	}
	fromJsonFs(v, jsonFs)

	for sc.Scan() {
		fs := &Fs{db: v.db}
		jsonFs, err := unmarshal(sc.Bytes())
		if err != nil {
			return fmt.Errorf("error unmarshinling Fs - %w", err)
		}
		if err := fromJsonFs(fs, jsonFs); err != nil {
			return fmt.Errorf("error fromJsonFs - %w", err)
		}

		paths, err := split(jsonFs.Path)
		if err != nil {
			return fmt.Errorf("error splitting path %v - %w", jsonFs.Path, err)
		}
		lastPath := len(paths) - 1
		parent, err := v.travelTo(paths[:lastPath], -1)
		if err != nil {
			return fmt.Errorf("error getting parent Fs %v - %w", jsonFs.Path, err)
		}
		// Check if the parent already has the child/name, if so, it must be a child
		parent, err = parent.travelTo(paths[lastPath:], -1)
		switch err {
		case nil:
			// the path already exists, so it MUST be a child
			parent.ref.child = fs
		case ErrNotFound: //nolint:errorlint
			// then this is a new path, so add the path
			parent.ref.children[fs.name] = fs
		default:
			return fmt.Errorf("error getting parent Fs %v - %w", jsonFs.Path, err)
		}
	}

	err = sc.Err()
	if err != nil {
		return fmt.Errorf("error reading db - %w", err)
	}

	err = os.Remove(dbFile)
	if err != nil {
		return fmt.Errorf("error removing db - %w", err)
	}

	return nil
}

type jsonFs struct {
	Path    string            `json:"path"`
	Child   bool              `json:"child"`
	Name    string            `json:"name"`
	Mode    uint32            `json:"mode"`
	ModTime time.Time         `json:"modTime"`
	Uid     string            `json:"uid"`
	Type    filetype.Filetype `json:"type"`
	Tags    map[string]any    `json:"tags"`
	Warning []string          `json:"warning"`
	Error   string            `json:"error"`
	Symlink string            `json:"symlink"`
	Size    int64             `json:"size"`
	MD5     string            `json:"md5"`
	SHA1    string            `json:"sha1"`
	SHA256  string            `json:"sha256"`
	SHA512  string            `json:"sha512"`
	Entropy float64           `json:"entropy"`
}

// ------------------------------JSON stuff--------------------------------
func toJsonFs(path string, child bool, n *Fs) jsonFs {
	tags := make(map[string]any)
	n.ref.tags.Range(func(key, value any) bool {
		tags[key.(string)] = value
		return true // Return true to continue iterating
	})

	err := ""
	if n.ref.err != nil {
		err = n.ref.err.Error()
	}
	warning := []string{}
	for _, e := range n.ref.warn {
		warning = append(warning, e.Error())
	}

	ref := n.ref
	return jsonFs{
		Path:    path,
		Child:   child,
		Name:    n.name,
		Mode:    uint32(n.mode),
		ModTime: n.modTime,
		Symlink: n.symlinkPath,
		Tags:    tags,
		Error:   err,
		Warning: warning,
		Uid:     ref.id,
		Type:    ref.typ,
		Size:    ref.size,
		MD5:     ref.md5,
		SHA1:    ref.sha1,
		SHA256:  ref.sha256,
		SHA512:  ref.sha512,
		Entropy: ref.entropy,
	}
}

func fromJsonFs(n *Fs, data jsonFs) error {
	n.name = data.Name
	n.mode = os.FileMode(data.Mode)
	n.modTime = data.ModTime
	n.symlinkPath = data.Symlink

	var updated bool
	n.ref, updated = n.db.updateIfDuplicate(&reference{
		id:       data.Uid,
		size:     data.Size,
		typ:      data.Type,
		md5:      data.MD5,
		sha1:     data.SHA1,
		sha256:   data.SHA256,
		sha512:   data.SHA512,
		entropy:  data.Entropy,
		children: make(map[string]*Fs),
	})

	if updated {
		return n.checkIfCircular(n.ref.sha512, true)
	}

	for k, v := range data.Tags {
		n.ref.tags.Store(k, v)
	}

	if data.Error != "" {
		n.ref.err = fmt.Errorf(data.Error)
		n.db.err = true
	}

	n.db.warn = n.db.warn || len(data.Warning) > 0
	for _, w := range data.Warning {
		n.ref.warn = append(n.ref.warn, errors.New(w))
	}
	return nil
}

func unmarshal(dataBytes []byte) (jsonFs, error) {
	toReturn := jsonFs{}
	return toReturn, json.Unmarshal(dataBytes, &toReturn)
}

package virtualfs

// FsError returns an error if the filesystem has an error
func (v *Fs) FsError() error {
	if v.db.err {
		return ErrInFilesystem
	}
	return nil
}

// FsWarning returns an error if the filesystem has a warning
func (v *Fs) FsWarning() error {
	if v.db.warn {
		return ErrInFilesystem
	}
	return nil
}

// Error sets the error for the Fs
func (n *Fs) Error(err error) {
	n.ref.err = err
	n.db.err = true
}

// Warning adds a warning to the Fs
func (n *Fs) Warning(warn error) {
	n.ref.warn = append(n.ref.warn, warn)
	n.db.warn = true
}

// TagS sets the tag with the given key to the given value
func (n *Fs) TagS(key string, value any) {
	n.ref.tags.Store(key, value)
}

// TagSIfBlank sets the tag with the given key to the given value if it is not already set
// returns ErrAlreadyExist if the tag already exists
func (n *Fs) TagSIfBlank(key string, value any) error {
	_, loaded := n.ref.tags.LoadOrStore(key, value)
	if loaded {
		return ErrAlreadyExist
	}
	return nil
}

// TagG returns the tag with the given key, return true if it exists, false if it does not
func (n *Fs) TagG(key string) (any, bool) {
	return n.ref.tags.Load(key)
}

// TagD deletes the tag with the given key, returns the tag
func (n *Fs) TagD(key string) (any, bool) {
	return n.ref.tags.LoadAndDelete(key)
}

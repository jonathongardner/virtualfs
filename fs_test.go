package virtualfs

import (
	"fmt"
	"io/fs"
	"os"
	"testing"
	"time"
)

// const barFile = "testdata/bar"
// const bazFile = "testdata/baz"

func createFile(v *Fs, path string, perm os.FileMode, modTime time.Time, content string) error {
	newV, err := v.Create(path, perm, modTime)
	if err != nil {
		return fmt.Errorf("failed to create virtual file  %w", err)
	}
	file, err := newV.CreateFile()
	if err != nil {
		return fmt.Errorf("failed to create file  %w", err)
	}
	_, err = file.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("failed to write to virtual file %w", err)
	}
	err = file.Close()
	if err != nil {
		return fmt.Errorf("failed to close virtual file %w", err)
	}

	return nil
}

func createChildFile(v *Fs, perm os.FileMode, modTime time.Time, content string) error {
	newV, err := v.CreateWithoutPath(perm, modTime)
	if err != nil {
		return fmt.Errorf("failed to create virtual file  %w", err)
	}
	file, err := newV.CreateFile()
	if err != nil {
		return fmt.Errorf("failed to create file  %w", err)
	}
	_, err = file.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("failed to write to virtual file %w", err)
	}
	err = file.Close()
	if err != nil {
		return fmt.Errorf("failed to close virtual file %w", err)
	}

	return nil
}

func TestVirtualOg(t *testing.T) {
	tmpDir(t, func(tmp string) {
		v, err := newFooFs(tmp)
		fatalfIfErr(t, err, "Failed to create virtual function")

		expected := []fileinfoTest{{"/", fooMod, fooTime, fooSha512, "application/octet-stream", "", emptyTags}}
		assertFiles(t, expected, v, "Initial")

		// add directory and make all paths needed
		v.MkdirP("/foo1/foo2", 0755, time1)
		expected = []fileinfoTest{
			{"/", fooMod, fooTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
		}
		assertFiles(t, expected, v, "after creating /foo1/foo2")

		// add another directory and make all paths needed
		v.MkdirP("/foo1/foo2/foo3/foo4", 0700, time2)
		expected = []fileinfoTest{
			{"/", fooMod, fooTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3/foo4", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
		}
		assertFiles(t, expected, v, "after creating foo1/foo2/foo3/foo4")
		assertTmpDirFileCount(t, 1, tmp, "after creating foo1/foo2/foo3/foo4")

		// Create a file
		err = createFile(v, "/foo1/foo2/foo3/bar", 0655, time3, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /foo1/foo2/foo3/bar")

		expected = []fileinfoTest{
			{"/", fooMod, fooTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3/bar", 0655, time3, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/foo1/foo2/foo3/foo4", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
		}
		assertFiles(t, expected, v, "after creating foo1/foo2/foo3/bar")
		assertTmpDirFileCount(t, 2, tmp, "after creating foo1/foo2/foo3/bar")

		// Create a symlink
		_, err = v.Symlink("/foo1/foo2/foo3/bar", "/foo1/foo2/symlink-bar", 0700, time1)
		fatalfIfErr(t, err, "failed to create symlink /foo1/foo2/symlink-bar")

		expected = []fileinfoTest{
			{"/", fooMod, fooTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3/bar", 0655, time3, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/foo1/foo2/foo3/foo4", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/symlink-bar", 0700 | fs.ModeSymlink, time1, "", "symlink/symlink", "/foo1/foo2/foo3/bar", emptyTags},
		}
		assertFiles(t, expected, v, "after creating /foo1/foo2/symlink-bar")
		assertTmpDirFileCount(t, 2, tmp, "after creating /foo1/foo2/symlink-bar")

		// Create a symlink to nowhere
		_, err = v.Symlink("/cool/beans/who-cares", "/foo1/foo2/symlink-nowhere", 0777, time2)
		fatalfIfErr(t, err, "failed to create symlink /foo1/foo2/symlink-nowhere")

		expected = []fileinfoTest{
			{"/", fooMod, fooTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3/bar", 0655, time3, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/foo1/foo2/foo3/foo4", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/symlink-bar", 0700 | fs.ModeSymlink, time1, "", "symlink/symlink", "/foo1/foo2/foo3/bar", emptyTags},
			{"/foo1/foo2/symlink-nowhere", 0777 | fs.ModeSymlink, time2, "", "symlink/symlink", "/cool/beans/who-cares", emptyTags},
		}
		assertFiles(t, expected, v, "after creating /foo1/foo2/symlink-nowhere")
		assertTmpDirFileCount(t, 2, tmp, "after creating /foo1/foo2/symlink-nowhere")

		// Create a hardlink
		_, err = v.Hardlink("/foo1/foo2/foo3/bar", "/foo1/foo2/hardlink-bar", 0700, time1)
		fatalfIfErr(t, err, "failed to create hardlink /foo1/foo2/hardlink-bar")

		expected = []fileinfoTest{
			{"/", fooMod, fooTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3/bar", 0655, time3, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/foo1/foo2/foo3/foo4", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/hardlink-bar", 0700, time1, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/foo1/foo2/symlink-bar", 0700 | fs.ModeSymlink, time1, "", "symlink/symlink", "/foo1/foo2/foo3/bar", emptyTags},
			{"/foo1/foo2/symlink-nowhere", 0777 | fs.ModeSymlink, time2, "", "symlink/symlink", "/cool/beans/who-cares", emptyTags},
		}
		assertFiles(t, expected, v, "after creating /foo1/foo2/hardlink-bar")
		assertTmpDirFileCount(t, 2, tmp, "after creating /foo1/foo2/hardlink-bar")

		_, err = v.Hardlink("/foo1/foo2/foo3/no-where", "/foo1/foo2/hardlink-bar-dont-get-created", 0700, time1)
		assertErr(t, ErrNotFound, err, "failed to error on bad hardlink /foo1/foo2/hardlink-bar-dont-get-created")
	})
}

func TestVirtualDontAllowCircular(t *testing.T) {
	tmpDir(t, func(tmp string) {
		v, err := newFooFs(tmp)
		fatalfIfErr(t, err, "Failed to create virtual function")

		err = createFile(v, "/foo-duplicate", 0655, time1, "foo")
		fatalfIfErr(t, err, "failed to create virtual file /foo-duplicate")

		newV, err := v.Stat("/foo-duplicate")
		fatalfIfErr(t, err, "failed to get /foo-duplicate")

		newV2, err := newV.CreateWithoutPath(0655, time1)
		fatalfIfErr(t, err, "failed to create virtual file without path")

		file, err := newV2.CreateFile()
		fatalfIfErr(t, err, "failed to create virtual file")

		_, err = file.Write([]byte("foo"))
		fatalfIfErr(t, err, "failed to write to virtual file")

		err = file.Close()
		assertErr(t, ErrCircularReference, err, "created a circular child virtual file")
	})
}

func TestVirtualBlankOg(t *testing.T) {
	tmpDir(t, func(tmp string) {
		f, err := os.Open(fooFile)
		fatalfIfErr(t, err, "Failed to open foo file")
		defer f.Close()

		v, err := NewFs(tmp, "", fooMod, fooTime, f)
		fatalfIfErr(t, err, "Failed to create virtual function")

		expected := []fileinfoTest{{"/", fooMod, fooTime, fooSha512, "application/octet-stream", "", emptyTags}}
		assertFiles(t, expected, v, "Initial")

		// add directory and make all paths needed
		v.MkdirP("/foo1/foo2", 0755, time1)
		expected = []fileinfoTest{
			{"/", fooMod, fooTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
		}
		assertFiles(t, expected, v, "after creating /foo1/foo2")

		// add another directory and make all paths needed
		v.MkdirP("/foo1/foo2/foo3/foo4", 0700, time2)
		expected = []fileinfoTest{
			{"/", fooMod, fooTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3/foo4", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
		}
		assertFiles(t, expected, v, "after creating foo1/foo2/foo3/foo4")
		assertTmpDirFileCount(t, 1, tmp, "after creating foo1/foo2/foo3/foo4")

		// Create a file
		err = createFile(v, "/foo1/foo2/foo3/bar", 0655, time3, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /foo1/foo2/foo3/bar")

		expected = []fileinfoTest{
			{"/", fooMod, fooTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3/bar", 0655, time3, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/foo1/foo2/foo3/foo4", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
		}
		assertFiles(t, expected, v, "after creating foo1/foo2/foo3/bar")
		assertTmpDirFileCount(t, 2, tmp, "after creating foo1/foo2/foo3/bar")

		// Create a symlink
		_, err = v.Symlink("/foo1/foo2/foo3/bar", "/foo1/foo2/symlink-bar", 0700, time1)
		fatalfIfErr(t, err, "failed to create symlink /foo1/foo2/symlink-bar")

		expected = []fileinfoTest{
			{"/", fooMod, fooTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3/bar", 0655, time3, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/foo1/foo2/foo3/foo4", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/symlink-bar", 0700 | fs.ModeSymlink, time1, "", "symlink/symlink", "/foo1/foo2/foo3/bar", emptyTags},
		}
		assertFiles(t, expected, v, "after creating /foo1/foo2/symlink-bar")
		assertTmpDirFileCount(t, 2, tmp, "after creating /foo1/foo2/symlink-bar")

		// Create a symlink to nowhere
		_, err = v.Symlink("/cool/beans/who-cares", "/foo1/foo2/symlink-nowhere", 0777, time2)
		fatalfIfErr(t, err, "failed to create symlink /foo1/foo2/symlink-nowhere")

		expected = []fileinfoTest{
			{"/", fooMod, fooTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3/bar", 0655, time3, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/foo1/foo2/foo3/foo4", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/symlink-bar", 0700 | fs.ModeSymlink, time1, "", "symlink/symlink", "/foo1/foo2/foo3/bar", emptyTags},
			{"/foo1/foo2/symlink-nowhere", 0777 | fs.ModeSymlink, time2, "", "symlink/symlink", "/cool/beans/who-cares", emptyTags},
		}
		assertFiles(t, expected, v, "after creating /foo1/foo2/symlink-nowhere")
		assertTmpDirFileCount(t, 2, tmp, "after creating /foo1/foo2/symlink-nowhere")
	})
}

func TestVirtualUsesReferencesForSameFile(t *testing.T) {
	tmpDir(t, func(tmp string) {
		v, err := newFooFs(tmp)
		fatalfIfErr(t, err, "failed to create virtual function")

		err = createFile(v, "/bar", 0655, time1, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /bar")

		err = createFile(v, "/baz", 0600, time2, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /baz")

		// should get added to both bar and baz since they are the same file
		newV, err := v.Stat("/baz")
		fatalfIfErr(t, err, "failed to create virtual filesystem from baz")

		err = createFile(newV, "/moreFoo", 0100, time3, "Hello, Foo!")
		fatalfIfErr(t, err, "failed to create virtual file /moreFoo from baz")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/bar", 0655, time1, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/bar/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/baz", 0600, time2, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/baz/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
		}
		assertFiles(t, expected, v, "after creating files in virtual from")
		assertTmpDirFileCount(t, 3, tmp, "after creating in virtual from")
	})
}

func TestVirtualDoesntAllowMovingOutsideFS(t *testing.T) {
	tmpDir(t, func(tmp string) {
		v, err := newFooFs(tmp)
		fatalfIfErr(t, err, "failed to create virtual function")

		_, err = v.Create("/bad/../not-cool/../../really", 0000, time1)
		assertErr(t, ErrOutsideFilesystem, err, "should error if trying to create outside filesystem 1")

		_, err = v.Create("bad/../not-cool/../../really", 0000, time1)
		assertErr(t, ErrOutsideFilesystem, err, "should error if trying to create outside filesystem 2")

		_, err = v.Create("../not-cool-either", 0000, time1)
		assertErr(t, ErrOutsideFilesystem, err, "should error if trying to create outside filesystem 3")

		_, err = v.Create("", 0000, time1)
		assertErr(t, ErrOutsideFilesystem, err, "should error if trying to create outside filesystem 4")

		err = createFile(v, "/bad/../okay/but-really-shouldnt-do-this", 0655, time1, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /bad/../okay/but-really-shouldnt-do-this")

		err = createFile(v, "bad/../okay/but-really-shouldnt-do-this-either", 0055, time2, "Hello, Foo!")
		fatalfIfErr(t, err, "failed to create virtual file bad/../okay/but-really-shouldnt-do-this-either")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/okay", 0655 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/okay/but-really-shouldnt-do-this", 0655, time1, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/okay/but-really-shouldnt-do-this-either", 0055, time2, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
		}
		assertFiles(t, expected, v, "after overwriting file with dir")
		assertTmpDirFileCount(t, 3, tmp, "after overwriting file with dir")
	})
}

func TestVirtualOverwriteFileWithDir(t *testing.T) {
	tmpDir(t, func(tmp string) {
		v, err := newFooFs(tmp)
		fatalfIfErr(t, err, "failed to create virtual function")

		err = createFile(v, "/bar", 0655, time1, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /bar")

		err = createFile(v, "/baz", 0600, time2, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /baz")

		err = createFile(v, "/bar/moreFoo", 0100, time3, "Hello, Foo!")
		fatalfIfErr(t, err, "failed to create virtual file /bar/moreFoo")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/bar", 0100 | fs.ModeDir, time3, "", "directory/directory", "", emptyTags},
			{"/bar/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/baz", 0600, time2, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
		}
		assertFiles(t, expected, v, "after overwriting file with dir")
		assertTmpDirFileCount(t, 3, tmp, "after overwriting file with dir")
	})
}

func TestVirtualFrom(t *testing.T) {
	tmpDir(t, func(tmp string) {
		v, err := newFooFs(tmp)
		fatalfIfErr(t, err, "failed to create virtual function")

		err = createFile(v, "/bar", 0655, time1, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /bar")

		newV, err := v.Stat("/bar")
		fatalfIfErr(t, err, "failed to create virtual from bar")

		newV.MkdirP("/", 0700, time1)  // shouldnt change anything cause root
		newV.MkdirP("./", 0700, time2) // shouldnt change anything cause root
		newV.MkdirP(".", 0700, time3)  // shouldnt change anything cause root

		err = createFile(newV, "/moreFoo", 0100, time3, "Hello, Foo!")
		fatalfIfErr(t, err, "failed to create /moreFoo from virtual file bar")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/bar", 0655, time1, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/bar/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
		}
		assertFiles(t, expected, v, "comparing")
	})
}

func TestTags(t *testing.T) {
	tmpDir(t, func(tmp string) {
		v, err := newFooFs(tmp)
		fatalfIfErr(t, err, "failed to create virtual function")

		_, ok := v.TagG("foo")
		assert(t, !ok, "foo value should not be set yet")

		v.TagS("foo", 47)
		val, ok := v.TagG("foo")
		assert(t, ok, "foo value should be set")
		assertEqual(t, 47, val, "should set key foo to 47")

		v.TagS("foo", 53)
		val, ok = v.TagG("foo")
		assert(t, ok, "foo value should still be set")
		assertEqual(t, 53, val, "should set key foo to 53")

		err = v.TagSIfBlank("foo", 7)
		assertErr(t, ErrAlreadyExist, err, "should return already set error")
		val, ok = v.TagG("foo")
		assert(t, ok, "foo value should still be set")
		assertEqual(t, 53, val, "key foo should still be set to 53")

		err = v.TagSIfBlank("bar", 7)
		assertEqual(t, nil, err, "should not return error since not set yet")
		val, ok = v.TagG("bar")
		assert(t, ok, "bar value should be set")
		assertEqual(t, 7, val, "should set key bar to 53")
	})
}

func TestWalk(t *testing.T) {
	tmpDir(t, func(tmp string) {
		v, err := newFooFs(tmp)
		fatalfIfErr(t, err, "failed to create virtual function")

		v.MkdirP("/foo1/foo2", 0755, time1)
		err = createFile(v, "/foo1/foo2/foo3/bar", 0655, time2, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /foo1/foo2/foo3/bar")

		assertPaths(t, []string{"/", "/foo1", "/foo1/foo2", "/foo1/foo2/foo3", "/foo1/foo2/foo3/bar"}, v, "after creating file structure")
	})
}

// this is needed for compressed files (i.e. foo.gz)
func TestSingleRootChild(t *testing.T) {
	tmpDir(t, func(tmp string) {
		f, err := os.Open(fooFile)
		fatalfIfErr(t, err, "Failed to open foo file")
		defer f.Close()

		// Use blank name otherwise the walked file will have a weird name
		v, err := NewFs(tmp, "", fooMod, fooTime, f)
		fatalfIfErr(t, err, "Failed to create virtual function")

		err = createChildFile(v, 0700, time2, helloWorldCompressed)
		fatalfIfErr(t, err, "failed to create child")

		newV, err := v.Stat("/")
		fatalfIfErr(t, err, "failed to create virtual filesystemfrom bar compressed")

		err = createFile(newV, "/moreFoo", 0100, time3, "Hello, Foo!")
		fatalfIfErr(t, err, "failed to create /moreFoo from virtual file bar")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/", 0700, time2, helloWorldCompressedSha512, "application/gzip", "", emptyTags},
			{"/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
		}
		assertFiles(t, expected, v, "comparing")

		fi, err := v.Stat("/")
		fatalfIfErr(t, err, "failed to get stat /bar from virtual file bar")
		assertEqual(t, "", fi.Name(), "should have name of original file for direct child")
		assertEqual(t, helloWorldCompressedSha512, fi.Sha512(), "should have foo sha512")

		fi, err = v.StatAt("/", 0)
		fatalfIfErr(t, err, "failed to get stat /bar at 0 from virtual file bar")
		assertEqual(t, "", fi.Name(), "should have name of original file for direct child")
		assertEqual(t, fooSha512, fi.Sha512(), "should have foo sha512")

		fi, err = v.StatAt("/", 1)
		fatalfIfErr(t, err, "failed to get stat /bar at 1 from virtual file bar")
		assertEqual(t, "", fi.Name(), "should have name of original file for direct child")
		assertEqual(t, helloWorldCompressedSha512, fi.Sha512(), "should have foo sha512")

		_, err = v.StatAt("/", 2)
		assertErr(t, ErrNotFound, err, "should error stat at not existing")
	})
}

// this is needed for compressed files (i.e. foo.gz)
func TestSingleChild(t *testing.T) {
	tmpDir(t, func(tmp string) {
		v, err := newFooFs(tmp)
		fatalfIfErr(t, err, "failed to create virtual function")

		err = createFile(v, "/bar", 0700, time2, helloWorldCompressed)
		fatalfIfErr(t, err, "failed to create child")

		newV, err := v.Stat("/bar")
		fatalfIfErr(t, err, "failed to create virtual from bar")

		err = createChildFile(newV, 0655, time1, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /bar")

		newVV, err := v.Stat("/bar")
		fatalfIfErr(t, err, "failed to create virtual filesystemfrom bar compressed")

		err = createFile(newVV, "/moreFoo", 0100, time3, "Hello, Foo!")
		fatalfIfErr(t, err, "failed to create /moreFoo from virtual file bar")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/bar", 0700, time2, helloWorldCompressedSha512, "application/gzip", "", emptyTags},
			{"/bar", 0655, time1, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/bar/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
		}
		assertFiles(t, expected, v, "comparing")

		fi, err := v.Stat("/bar")
		fatalfIfErr(t, err, "failed to get stat /bar from virtual file bar")
		assertEqual(t, "bar", fi.Name(), "should have name of original file for direct child")
		assertEqual(t, helloWorldSha512, fi.Sha512(), "should have foo sha512")

		fi, err = v.StatAt("/bar", 0)
		fatalfIfErr(t, err, "failed to get stat /bar at 0 from virtual file bar")
		assertEqual(t, "bar", fi.Name(), "should have name of original file for direct child")
		assertEqual(t, helloWorldCompressedSha512, fi.Sha512(), "should have foo sha512")

		fi, err = v.StatAt("/bar", 1)
		fatalfIfErr(t, err, "failed to get stat /bar at 1 from virtual file bar")
		assertEqual(t, "bar", fi.Name(), "should have name of original file for direct child")
		assertEqual(t, helloWorldSha512, fi.Sha512(), "should have foo sha512")

		_, err = v.StatAt("/bar", 2)
		assertErr(t, ErrNotFound, err, "should error stat at not existing")

		_, err = v.CreateWithoutPath(0000, time1)
		assertErr(t, ErrAlreadyHasChildren, err, "should error if trying to create child on one with children")

		_, err = newV.Create("/shouldFail", 0000, time1)
		assertErr(t, ErrAlreadyHasChild, err, "should error if trying to create children on one with child")
	})
}

func TestVirtaulWithDir(t *testing.T) {
	tmpDir(t, func(tmp string) {
		v, err := newTestFolderFs(tmp)
		fatalfIfErr(t, err, "failed to create virtual function")

		expected := []fileinfoTest{
			{"/", testMod, ignoreTime, "", "directory/directory", "", emptyTags},
		}
		assertFiles(t, expected, v, "comparing")
		assertTmpDirFileCount(t, 0, tmp, "comparing")
	})
}

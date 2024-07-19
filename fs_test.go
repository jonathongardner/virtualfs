package virtualfs

import (
	"fmt"
	"os"
	"testing"
)

const fooFile = "testdata/foo"
const fooSha512 = "0f5623276549769a63c79ca20fc573518685819fe82b39f43a3e7cf709c8baa16524daa95e006e81f7267700a88adee8a6209201be960a10c81c35ff3547e3b7"

// const barFile = "testdata/bar"
// const bazFile = "testdata/baz"

func createFile(v *Fs, path string, perm os.FileMode, content string) error {
	file, err := v.Create(path, perm)
	if err != nil {
		return fmt.Errorf("failed to create virtual file  %v", err)
	}
	_, err = file.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("failed to write to virtual file %v", err)
	}
	err = file.Close()
	if err != nil {
		return fmt.Errorf("failed to close virtual file %v", err)
	}

	return nil
}

func TestVirtual(t *testing.T) {
	myT := NewMyT("Test Virtual", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile)
		myT.FatalfIfErr(err, "Failed to create virtual function")

		expected := []fileinfoTest{{"/", 0664, fooSha512, "application/octet-stream"}}
		myT.AssertFiles(expected, v, "Initial")

		v.MkdirP("/foo1/foo2", 0755)
		expected = []fileinfoTest{
			{"/", 0664, fooSha512, "application/octet-stream"},
			{"/foo1", 0755, "", "directory/directory"},
			{"/foo1/foo2", 0755, "", "directory/directory"},
		}
		myT.AssertFiles(expected, v, "after creating /foo1/foo2")

		v.MkdirP("/foo1/foo2/foo3/foo4", 0700)
		expected = []fileinfoTest{
			{"/", 0664, fooSha512, "application/octet-stream"},
			{"/foo1", 0755, "", "directory/directory"},
			{"/foo1/foo2", 0755, "", "directory/directory"},
			{"/foo1/foo2/foo3", 0700, "", "directory/directory"},
			{"/foo1/foo2/foo3/foo4", 0700, "", "directory/directory"},
		}
		myT.AssertFiles(expected, v, "after creating foo1/foo2/foo3/foo4")
		myT.AssertTmpDirFileCount(1, tmp, "after creating foo1/foo2/foo3/foo4")

		err = createFile(v, "/foo1/foo2/foo3/bar", 0655, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo1/foo2/foo3/bar")

		expected = []fileinfoTest{
			{"/", 0664, fooSha512, "application/octet-stream"},
			{"/foo1", 0755, "", "directory/directory"},
			{"/foo1/foo2", 0755, "", "directory/directory"},
			{"/foo1/foo2/foo3", 0700, "", "directory/directory"},
			{"/foo1/foo2/foo3/foo4", 0700, "", "directory/directory"},
			{"/foo1/foo2/foo3/bar", 0655, "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387", "text/plain; charset=utf-8"},
		}
		myT.AssertFiles(expected, v, "after creating foo1/foo2/foo3/bar")
		myT.AssertTmpDirFileCount(2, tmp, "after creating foo1/foo2/foo3/bar")
	})
}

func TestVirtualUsesReferencesForSameFile(t *testing.T) {
	myT := NewMyT("Test virtual uses references for same file", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile)
		myT.FatalfIfErr(err, "failed to create virtual function")

		err = createFile(v, "/bar", 0655, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /bar")

		err = createFile(v, "/baz", 0600, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /baz")

		// should get added to both bar and baz since they are the same file
		newV, err := v.FsFrom("/baz")
		myT.FatalfIfErr(err, "failed to create virtual from baz")

		err = createFile(newV, "/moreFoo", 0100, "Hello, Foo!")
		myT.FatalfIfErr(err, "failed to create virtual file /moreFoo from baz")

		expected := []fileinfoTest{
			{"/", 0664, fooSha512, "application/octet-stream"},
			{"/bar", 0655, "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387", "text/plain; charset=utf-8"},
			{"/bar/moreFoo", 0100, "9b617e0675ac2ede198cfacddf0b283d378a2cee8e72e551a1ae5400cdb9a46792556187e4d2fdbedece0f0021a6b1f74a6b460b62966ef68025abf75fb7df7a", "text/plain; charset=utf-8"},
			{"/baz", 0600, "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387", "text/plain; charset=utf-8"},
			{"/baz/moreFoo", 0100, "9b617e0675ac2ede198cfacddf0b283d378a2cee8e72e551a1ae5400cdb9a46792556187e4d2fdbedece0f0021a6b1f74a6b460b62966ef68025abf75fb7df7a", "text/plain; charset=utf-8"},
		}
		myT.AssertFiles(expected, v, "after creating files in virtual from")
		myT.AssertTmpDirFileCount(3, tmp, "after creating in virtual from")
	})
}

func TestVirtualOverwriteFileWithDir(t *testing.T) {
	myT := NewMyT("Test virtual overwrites file with dir", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile)
		myT.FatalfIfErr(err, "failed to create virtual function")

		err = createFile(v, "/bar", 0655, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /bar")

		err = createFile(v, "/baz", 0600, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /baz")

		err = createFile(v, "/bar/moreFoo", 0100, "Hello, Foo!")
		myT.FatalfIfErr(err, "failed to create virtual file /bar/moreFoo")

		expected := []fileinfoTest{
			{"/", 0664, fooSha512, "application/octet-stream"},
			{"/bar", 0100, "", "directory/directory"},
			{"/bar/moreFoo", 0100, "9b617e0675ac2ede198cfacddf0b283d378a2cee8e72e551a1ae5400cdb9a46792556187e4d2fdbedece0f0021a6b1f74a6b460b62966ef68025abf75fb7df7a", "text/plain; charset=utf-8"},
			{"/baz", 0600, "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387", "text/plain; charset=utf-8"},
		}
		myT.AssertFiles(expected, v, "after overwriting file with dir")
		myT.AssertTmpDirFileCount(3, tmp, "after overwriting file with dir")
	})
}

func TestVirtualFrom(t *testing.T) {
	myT := NewMyT("Test virtual from", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile)
		myT.FatalfIfErr(err, "failed to create virtual function")

		err = createFile(v, "/bar", 0655, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /bar")

		newV, err := v.FsFrom("/bar")
		myT.FatalfIfErr(err, "failed to create virtual from bar")

		newV.MkdirP("/", 0700)  // shouldnt change anything cause root
		newV.MkdirP("./", 0700) // shouldnt change anything cause root
		newV.MkdirP(".", 0700)  // shouldnt change anything cause root

		err = createFile(newV, "/moreFoo", 0100, "Hello, Foo!")
		myT.FatalfIfErr(err, "failed to create /moreFoo from virtual file bar")

		expected := []fileinfoTest{
			{"/", 0664, fooSha512, "application/octet-stream"},
			{"/bar", 0655, "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387", "text/plain; charset=utf-8"},
			{"/bar/moreFoo", 0100, "9b617e0675ac2ede198cfacddf0b283d378a2cee8e72e551a1ae5400cdb9a46792556187e4d2fdbedece0f0021a6b1f74a6b460b62966ef68025abf75fb7df7a", "text/plain; charset=utf-8"},
		}
		myT.AssertFiles(expected, v, "comparing")
	})
}

func TestWalk(t *testing.T) {
	myT := NewMyT("Test Walk", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile)
		myT.FatalfIfErr(err, "failed to create virtual function")

		v.MkdirP("/foo1/foo2", 0755)
		err = createFile(v, "/foo1/foo2/foo3/bar", 0655, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo1/foo2/foo3/bar")

		myT.AssertPaths([]string{"/", "/foo1", "/foo1/foo2", "/foo1/foo2/foo3", "/foo1/foo2/foo3/bar"}, v, "after creating file structure")
	})
}

package virtualfs

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// const barFile = "testdata/bar"
// const bazFile = "testdata/baz"

func createFile(v *Fs, path string, perm os.FileMode, modTime time.Time, content string) error {
	file, err := v.Create(path, perm, modTime)
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

		expected := []fileinfoTest{{"/", 0664, ignoreTime, fooSha512, "application/octet-stream", ""}}
		myT.AssertFiles(expected, v, "Initial")

		// add directory and make all paths needed
		v.MkdirP("/foo1/foo2", 0755, time1)
		expected = []fileinfoTest{
			{"/", 0664, ignoreTime, fooSha512, "application/octet-stream", ""},
			{"/foo1", 0755, time1, "", "directory/directory", ""},
			{"/foo1/foo2", 0755, time1, "", "directory/directory", ""},
		}
		myT.AssertFiles(expected, v, "after creating /foo1/foo2")

		// add another directory and make all paths needed
		v.MkdirP("/foo1/foo2/foo3/foo4", 0700, time2)
		expected = []fileinfoTest{
			{"/", 0664, ignoreTime, fooSha512, "application/octet-stream", ""},
			{"/foo1", 0755, time1, "", "directory/directory", ""},
			{"/foo1/foo2", 0755, time1, "", "directory/directory", ""},
			{"/foo1/foo2/foo3", 0700, time2, "", "directory/directory", ""},
			{"/foo1/foo2/foo3/foo4", 0700, time2, "", "directory/directory", ""},
		}
		myT.AssertFiles(expected, v, "after creating foo1/foo2/foo3/foo4")
		myT.AssertTmpDirFileCount(1, tmp, "after creating foo1/foo2/foo3/foo4")

		// Create a file
		err = createFile(v, "/foo1/foo2/foo3/bar", 0655, time3, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo1/foo2/foo3/bar")

		expected = []fileinfoTest{
			{"/", 0664, ignoreTime, fooSha512, "application/octet-stream", ""},
			{"/foo1", 0755, time1, "", "directory/directory", ""},
			{"/foo1/foo2", 0755, time1, "", "directory/directory", ""},
			{"/foo1/foo2/foo3", 0700, time2, "", "directory/directory", ""},
			{"/foo1/foo2/foo3/bar", 0655, time3, helloWorldSha512, "text/plain; charset=utf-8", ""},
			{"/foo1/foo2/foo3/foo4", 0700, time2, "", "directory/directory", ""},
		}
		myT.AssertFiles(expected, v, "after creating foo1/foo2/foo3/bar")
		myT.AssertTmpDirFileCount(2, tmp, "after creating foo1/foo2/foo3/bar")

		// Create a symlink
		err = v.Symlink("/foo1/foo2/foo3/bar", "/foo1/foo2/symlink-bar", 0700, time1)
		myT.FatalfIfErr(err, "failed to create symlink /foo1/foo2/symlink-bar")

		expected = []fileinfoTest{
			{"/", 0664, ignoreTime, fooSha512, "application/octet-stream", ""},
			{"/foo1", 0755, time1, "", "directory/directory", ""},
			{"/foo1/foo2", 0755, time1, "", "directory/directory", ""},
			{"/foo1/foo2/foo3", 0700, time2, "", "directory/directory", ""},
			{"/foo1/foo2/foo3/bar", 0655, time3, helloWorldSha512, "text/plain; charset=utf-8", ""},
			{"/foo1/foo2/foo3/foo4", 0700, time2, "", "directory/directory", ""},
			{"/foo1/foo2/symlink-bar", 0700, time1, "", "symlink/symlink", "/foo1/foo2/foo3/bar"},
		}
		myT.AssertFiles(expected, v, "after creating /foo1/foo2/symlink-bar")
		myT.AssertTmpDirFileCount(2, tmp, "after creating /foo1/foo2/symlink-bar")

		// Create a symlink to nowhere
		err = v.Symlink("/cool/beans/who-cares", "/foo1/foo2/symlink-nowhere", 0777, time2)
		myT.FatalfIfErr(err, "failed to create symlink /foo1/foo2/symlink-nowhere")

		expected = []fileinfoTest{
			{"/", 0664, ignoreTime, fooSha512, "application/octet-stream", ""},
			{"/foo1", 0755, time1, "", "directory/directory", ""},
			{"/foo1/foo2", 0755, time1, "", "directory/directory", ""},
			{"/foo1/foo2/foo3", 0700, time2, "", "directory/directory", ""},
			{"/foo1/foo2/foo3/bar", 0655, time3, helloWorldSha512, "text/plain; charset=utf-8", ""},
			{"/foo1/foo2/foo3/foo4", 0700, time2, "", "directory/directory", ""},
			{"/foo1/foo2/symlink-bar", 0700, time1, "", "symlink/symlink", "/foo1/foo2/foo3/bar"},
			{"/foo1/foo2/symlink-nowhere", 0777, time2, "", "symlink/symlink", "/cool/beans/who-cares"},
		}
		myT.AssertFiles(expected, v, "after creating /foo1/foo2/symlink-nowhere")
		myT.AssertTmpDirFileCount(2, tmp, "after creating /foo1/foo2/symlink-nowhere")
	})
}

func TestVirtualUsesReferencesForSameFile(t *testing.T) {
	myT := NewMyT("Test virtual uses references for same file", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile)
		myT.FatalfIfErr(err, "failed to create virtual function")

		err = createFile(v, "/bar", 0655, time1, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /bar")

		err = createFile(v, "/baz", 0600, time2, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /baz")

		// should get added to both bar and baz since they are the same file
		newV, err := v.FsFrom("/baz")
		myT.FatalfIfErr(err, "failed to create virtual from baz")

		err = createFile(newV, "/moreFoo", 0100, time3, "Hello, Foo!")
		myT.FatalfIfErr(err, "failed to create virtual file /moreFoo from baz")

		expected := []fileinfoTest{
			{"/", 0664, ignoreTime, fooSha512, "application/octet-stream", ""},
			{"/bar", 0655, time1, helloWorldSha512, "text/plain; charset=utf-8", ""},
			{"/bar/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", ""},
			{"/baz", 0600, time2, helloWorldSha512, "text/plain; charset=utf-8", ""},
			{"/baz/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", ""},
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

		err = createFile(v, "/bar", 0655, time1, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /bar")

		err = createFile(v, "/baz", 0600, time2, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /baz")

		err = createFile(v, "/bar/moreFoo", 0100, time3, "Hello, Foo!")
		myT.FatalfIfErr(err, "failed to create virtual file /bar/moreFoo")

		expected := []fileinfoTest{
			{"/", 0664, ignoreTime, fooSha512, "application/octet-stream", ""},
			{"/bar", 0100, time3, "", "directory/directory", ""},
			{"/bar/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", ""},
			{"/baz", 0600, time2, helloWorldSha512, "text/plain; charset=utf-8", ""},
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

		err = createFile(v, "/bar", 0655, time1, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /bar")

		newV, err := v.FsFrom("/bar")
		myT.FatalfIfErr(err, "failed to create virtual from bar")

		newV.MkdirP("/", 0700, time1)  // shouldnt change anything cause root
		newV.MkdirP("./", 0700, time2) // shouldnt change anything cause root
		newV.MkdirP(".", 0700, time3)  // shouldnt change anything cause root

		err = createFile(newV, "/moreFoo", 0100, time3, "Hello, Foo!")
		myT.FatalfIfErr(err, "failed to create /moreFoo from virtual file bar")

		expected := []fileinfoTest{
			{"/", 0664, ignoreTime, fooSha512, "application/octet-stream", ""},
			{"/bar", 0655, time1, helloWorldSha512, "text/plain; charset=utf-8", ""},
			{"/bar/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", ""},
		}
		myT.AssertFiles(expected, v, "comparing")
	})
}

func TestWalk(t *testing.T) {
	myT := NewMyT("Test Walk", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile)
		myT.FatalfIfErr(err, "failed to create virtual function")

		v.MkdirP("/foo1/foo2", 0755, time1)
		err = createFile(v, "/foo1/foo2/foo3/bar", 0655, time2, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo1/foo2/foo3/bar")

		myT.AssertPaths([]string{"/", "/foo1", "/foo1/foo2", "/foo1/foo2/foo3", "/foo1/foo2/foo3/bar"}, v, "after creating file structure")
	})
}

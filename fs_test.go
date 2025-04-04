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

func createChildFile(v *Fs, perm os.FileMode, modTime time.Time, content string) error {
	file, err := v.CreateChild(perm, modTime)
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
		v, err := NewFs(tmp, fooFile, false)
		myT.FatalfIfErr(err, "Failed to create virtual function")

		expected := []fileinfoTest{{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags}}
		myT.AssertFiles(expected, v, "Initial")

		// add directory and make all paths needed
		v.MkdirP("/foo1/foo2", 0755, time1)
		expected = []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
		}
		myT.AssertFiles(expected, v, "after creating /foo1/foo2")

		// add another directory and make all paths needed
		v.MkdirP("/foo1/foo2/foo3/foo4", 0700, time2)
		expected = []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3/foo4", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
		}
		myT.AssertFiles(expected, v, "after creating foo1/foo2/foo3/foo4")
		myT.AssertTmpDirFileCount(1, tmp, "after creating foo1/foo2/foo3/foo4")

		// Create a file
		err = createFile(v, "/foo1/foo2/foo3/bar", 0655, time3, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo1/foo2/foo3/bar")

		expected = []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3/bar", 0655, time3, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/foo1/foo2/foo3/foo4", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
		}
		myT.AssertFiles(expected, v, "after creating foo1/foo2/foo3/bar")
		myT.AssertTmpDirFileCount(2, tmp, "after creating foo1/foo2/foo3/bar")

		// Create a symlink
		err = v.Symlink("/foo1/foo2/foo3/bar", "/foo1/foo2/symlink-bar", 0700, time1)
		myT.FatalfIfErr(err, "failed to create symlink /foo1/foo2/symlink-bar")

		expected = []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3/bar", 0655, time3, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/foo1/foo2/foo3/foo4", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/symlink-bar", 0700 | fs.ModeSymlink, time1, "", "symlink/symlink", "/foo1/foo2/foo3/bar", emptyTags},
		}
		myT.AssertFiles(expected, v, "after creating /foo1/foo2/symlink-bar")
		myT.AssertTmpDirFileCount(2, tmp, "after creating /foo1/foo2/symlink-bar")

		// Create a symlink to nowhere
		err = v.Symlink("/cool/beans/who-cares", "/foo1/foo2/symlink-nowhere", 0777, time2)
		myT.FatalfIfErr(err, "failed to create symlink /foo1/foo2/symlink-nowhere")

		expected = []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo1", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/foo3/bar", 0655, time3, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/foo1/foo2/foo3/foo4", 0700 | fs.ModeDir, time2, "", "directory/directory", "", emptyTags},
			{"/foo1/foo2/symlink-bar", 0700 | fs.ModeSymlink, time1, "", "symlink/symlink", "/foo1/foo2/foo3/bar", emptyTags},
			{"/foo1/foo2/symlink-nowhere", 0777 | fs.ModeSymlink, time2, "", "symlink/symlink", "/cool/beans/who-cares", emptyTags},
		}
		myT.AssertFiles(expected, v, "after creating /foo1/foo2/symlink-nowhere")
		myT.AssertTmpDirFileCount(2, tmp, "after creating /foo1/foo2/symlink-nowhere")
	})
}

func TestVirtualUsesReferencesForSameFile(t *testing.T) {
	myT := NewMyT("Test virtual uses references for same file", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile, false)
		myT.FatalfIfErr(err, "failed to create virtual function")

		err = createFile(v, "/bar", 0655, time1, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /bar")

		err = createFile(v, "/baz", 0600, time2, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /baz")

		// should get added to both bar and baz since they are the same file
		newV, err := v.FsFrom("/baz")
		myT.FatalfIfErr(err, "failed to create virtual filesystem from baz")

		err = createFile(newV, "/moreFoo", 0100, time3, "Hello, Foo!")
		myT.FatalfIfErr(err, "failed to create virtual file /moreFoo from baz")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/bar", 0655, time1, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/bar/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/baz", 0600, time2, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/baz/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
		}
		myT.AssertFiles(expected, v, "after creating files in virtual from")
		myT.AssertTmpDirFileCount(3, tmp, "after creating in virtual from")
	})
}

func TestVirtualDoesntAllowMovingOutsideFS(t *testing.T) {
	myT := NewMyT("Test virtual doesnt allow moving outside filesystem", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile, false)
		myT.FatalfIfErr(err, "failed to create virtual function")

		_, err = v.Create("/bad/../not-cool/../../really", 0000, time1)
		myT.AssertErr(ErrOutsideFilesystem, err, "should error if trying to create outside filesystem 1")

		_, err = v.Create("bad/../not-cool/../../really", 0000, time1)
		myT.AssertErr(ErrOutsideFilesystem, err, "should error if trying to create outside filesystem 2")

		_, err = v.Create("../not-cool-either", 0000, time1)
		myT.AssertErr(ErrOutsideFilesystem, err, "should error if trying to create outside filesystem 3")

		err = createFile(v, "/bad/../okay/but-really-shouldnt-do-this", 0655, time1, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /bad/../okay/but-really-shouldnt-do-this")

		err = createFile(v, "bad/../okay/but-really-shouldnt-do-this-either", 0055, time2, "Hello, Foo!")
		myT.FatalfIfErr(err, "failed to create virtual file bad/../okay/but-really-shouldnt-do-this-either")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/okay", 0655 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/okay/but-really-shouldnt-do-this", 0655, time1, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/okay/but-really-shouldnt-do-this-either", 0055, time2, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
		}
		myT.AssertFiles(expected, v, "after overwriting file with dir")
		myT.AssertTmpDirFileCount(3, tmp, "after overwriting file with dir")
	})
}

func TestVirtualOverwriteFileWithDir(t *testing.T) {
	myT := NewMyT("Test virtual overwrites file with dir", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile, false)
		myT.FatalfIfErr(err, "failed to create virtual function")

		err = createFile(v, "/bar", 0655, time1, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /bar")

		err = createFile(v, "/baz", 0600, time2, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /baz")

		err = createFile(v, "/bar/moreFoo", 0100, time3, "Hello, Foo!")
		myT.FatalfIfErr(err, "failed to create virtual file /bar/moreFoo")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/bar", 0100 | fs.ModeDir, time3, "", "directory/directory", "", emptyTags},
			{"/bar/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/baz", 0600, time2, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
		}
		myT.AssertFiles(expected, v, "after overwriting file with dir")
		myT.AssertTmpDirFileCount(3, tmp, "after overwriting file with dir")
	})
}

func TestVirtualFrom(t *testing.T) {
	myT := NewMyT("Test virtual from", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile, false)
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
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/bar", 0655, time1, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/bar/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
		}
		myT.AssertFiles(expected, v, "comparing")
	})
}

func TestTags(t *testing.T) {
	myT := NewMyT("Test Tags", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile, false)
		myT.FatalfIfErr(err, "failed to create virtual function")

		_, ok := v.TagG("foo")
		myT.Assert(!ok, "foo value should not be set yet")

		v.TagS("foo", 47)
		val, ok := v.TagG("foo")
		myT.Assert(ok, "foo value should be set")
		myT.AssertEqual(47, val, "should set key foo to 47")

		v.TagS("foo", 53)
		val, ok = v.TagG("foo")
		myT.Assert(ok, "foo value should still be set")
		myT.AssertEqual(53, val, "should set key foo to 53")

		err = v.TagSIfBlank("foo", 7)
		myT.AssertErr(ErrAlreadyExist, err, "should return already set error")
		val, ok = v.TagG("foo")
		myT.Assert(ok, "foo value should still be set")
		myT.AssertEqual(53, val, "key foo should still be set to 53")

		err = v.TagSIfBlank("bar", 7)
		myT.AssertEqual(nil, err, "should not return error since not set yet")
		val, ok = v.TagG("bar")
		myT.Assert(ok, "bar value should be set")
		myT.AssertEqual(7, val, "should set key bar to 53")
	})
}

func TestWalk(t *testing.T) {
	myT := NewMyT("Test Walk", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile, false)
		myT.FatalfIfErr(err, "failed to create virtual function")

		v.MkdirP("/foo1/foo2", 0755, time1)
		err = createFile(v, "/foo1/foo2/foo3/bar", 0655, time2, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo1/foo2/foo3/bar")

		myT.AssertPaths([]string{"/", "/foo1", "/foo1/foo2", "/foo1/foo2/foo3", "/foo1/foo2/foo3/bar"}, v, "after creating file structure")
	})
}

// this is needed for compressed files (i.e. foo.gz)
func TestSingleChild(t *testing.T) {
	myT := NewMyT("Test virtual Single Child", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, fooFile, false)
		myT.FatalfIfErr(err, "failed to create virtual function")

		err = createFile(v, "/bar", 0655, time1, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /bar")

		newV, err := v.FsFrom("/bar")
		myT.FatalfIfErr(err, "failed to create virtual from bar")

		err = createChildFile(newV, 0700, time2, "sZ�f�H�����/�IQ����")
		myT.FatalfIfErr(err, "failed to create child")

		newVV, err := v.FsFrom("/bar")
		myT.FatalfIfErr(err, "failed to create virtual filesystemfrom bar compressed")

		err = createFile(newVV, "/moreFoo", 0100, time3, "Hello, Foo!")
		myT.FatalfIfErr(err, "failed to create /moreFoo from virtual file bar")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/bar", 0655, time1, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/bar", 0700, time2, helloWorldCompressedSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/bar/moreFoo", 0100, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
		}
		myT.AssertFiles(expected, v, "comparing")

		fi, err := v.Stat("/bar")
		myT.FatalfIfErr(err, "failed to get stat /bar from virtual file bar")
		myT.AssertEqual("bar", fi.Name(), "should have name of original file for direct child")
		myT.AssertEqual(helloWorldCompressedSha512, fi.Sha512(), "should have foo sha512")

		fi, err = v.StatAt("/bar", 0)
		myT.FatalfIfErr(err, "failed to get stat /bar at 0 from virtual file bar")
		myT.AssertEqual("bar", fi.Name(), "should have name of original file for direct child")
		myT.AssertEqual(helloWorldSha512, fi.Sha512(), "should have foo sha512")

		fi, err = v.StatAt("/bar", 1)
		myT.FatalfIfErr(err, "failed to get stat /bar at 1 from virtual file bar")
		myT.AssertEqual("bar", fi.Name(), "should have name of original file for direct child")
		myT.AssertEqual(helloWorldCompressedSha512, fi.Sha512(), "should have foo sha512")

		_, err = v.StatAt("/bar", 2)
		myT.AssertErr(ErrNotFound, err, "should error stat at not existing")

		_, err = v.CreateChild(0000, time1)
		myT.AssertErr(ErrAlreadyHasChildren, err, "should error if trying to create child on one with children")

		_, err = newV.Create("/shouldFail", 0000, time1)
		myT.AssertErr(ErrAlreadyHasChild, err, "should error if trying to create children on one with child")
	})
}

func TestVirtaulWithDir(t *testing.T) {
	myT := NewMyT("Test virtual Single Child", t)
	myT.TmpDir(func(tmp string) {
		v, err := NewFs(tmp, testFolder, false)
		myT.FatalfIfErr(err, "failed to create virtual function")

		expected := []fileinfoTest{
			{"/", 0775 | os.ModeDir, ignoreTime, "", "directory/directory", "", emptyTags},
			{"/bar", barMod, ignoreTime, barSha512, "application/gzip", "", emptyTags},
			{"/foo", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/more", 0775 | os.ModeDir, ignoreTime, "", "directory/directory", "", emptyTags},
			{"/more/baz", bazMod, ignoreTime, bazSha512, "application/octet-stream", "", emptyTags},
			{"/more/foo", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
		}
		myT.AssertFiles(expected, v, "comparing")
		myT.AssertTmpDirFileCount(3, tmp, "comparing")
	})
}

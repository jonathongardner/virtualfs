package virtualfs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestVirtualClose(t *testing.T) {
	myT := NewMyT("Test Virtual Close", t)
	myT.TmpDir(func(tmp string) {
		//------------ Setup Filesystem
		v, err := NewFs(tmp, fooFile, false)
		myT.FatalfIfErr(err, "Failed to create virtual function")

		v.TagS("foo", "bar")
		v.TagS("baz", 47)

		err = v.MkdirP("/foo", 0755, time1)
		myT.FatalfIfErr(err, "failed to create virtual folder /foo")

		err = createFile(v, "/foo/bar", 0655, time2, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo/bar")

		fooV, err := v.FsFrom("/foo/bar")
		fooV.TagS("foo2", "bar2")
		fooV.TagS("processed", true)
		myT.FatalfIfErr(err, "failed to create virtual filesystem /foo/bar")
		fooV.Warning(fmt.Errorf("yikes! somthing kinda whent wrong"))

		err = v.Symlink("/foo/bar", "/foo/bar-symlink", 0777, time3)
		myT.FatalfIfErr(err, "failed to create symlink /foo/bar-symlink")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", map[any]any{"foo": "bar", "baz": 47}},
			{"/foo", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo/bar", 0655, time2, helloWorldSha512, "text/plain; charset=utf-8", "", map[any]any{"foo2": "bar2", "processed": true}},
			{"/foo/bar-symlink", 0777 | fs.ModeSymlink, time3, "", "symlink/symlink", "/foo/bar", emptyTags},
		}
		myT.AssertFiles(expected, v, "after adding files in virtual file system")
		myT.AssertTmpDirFileCount(2, tmp, "after adding files in virtual file system")
		myT.Assert(v.ProcessError() == nil, "should NOT have error if not set")
		myT.AssertErr(v.ProcessWarning(), ErrInFilesystem, "after setting warning")

		//------------ Close
		err = v.Close()
		myT.FatalfIfErr(err, "error closing file /foo1/foo2/foo3/bar")
		myT.AssertTmpDirFileCount(3, tmp, "after closing virtual file system")

		//------------ Make sure DB file exists
		_, err = os.Stat(filepath.Join(tmp, "fin.db"))
		myT.AssertEqual(err, nil, "expected db file to exist after closing")

		//------------ Expect error if try to make changes
		err = v.MkdirP("/foo/should-fail-folder", 0755, time1)
		myT.AssertEqual(err, ErrClosed, "expected error creating virtual folder /foo/should-fail-folder after closing")

		_, err = v.Create("/foo/should-fail-file", 0655, time1)
		myT.AssertEqual(err, ErrClosed, "expected error creating virtual file /foo/should-fail-file after closing")

		// Create a symlink
		err = v.Symlink("/foo/bar", "/foo/another-symlink", 0700, time1)
		myT.AssertEqual(err, ErrClosed, "expected error creating symlink /foo/another-symlink after closing")

		//------------ Load folder and make sure it works
		newV, err := NewFsFromDb(tmp)
		myT.FatalfIfErr(err, "failed to create filesystem from dir")
		myT.AssertFiles(expected, newV, "after loading files in virtual from")
		myT.AssertTmpDirFileCount(2, tmp, "after loading fs should delete manifest")

		err = newV.MkdirP("/foo/new-folder", 0155, time1)
		myT.FatalfIfErr(err, "failed to create virtual folder /foo/new-folder")

		err = createFile(newV, "/foo/duplicate", 0200, time2, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo/new-file")

		// Create a file
		err = createFile(newV, "/foo/new-file", 0655, time3, "Hello, Foo!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo/new-file")

		newExpected := append(
			expected,
			fileinfoTest{"/foo/duplicate", 0200, time2, helloWorldSha512, "text/plain; charset=utf-8", "", map[any]any{"foo2": "bar2", "processed": true}},
			fileinfoTest{"/foo/new-file", 0655, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
			fileinfoTest{"/foo/new-folder", 0155 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
		)
		myT.AssertFiles(newExpected, newV, "after creating files in virtual from")
		myT.AssertTmpDirFileCount(3, tmp, "after creating in virtual from")
		myT.Assert(newV.ProcessError() == nil, "should NOT have error on load if it wasnt set before save")
		myT.AssertErr(newV.ProcessWarning(), ErrInFilesystem, "should have warning when loading")
	})
}

func TestVirtualCloseWithErr(t *testing.T) {
	myT := NewMyT("Test Virtual Close", t)
	myT.TmpDir(func(tmp string) {
		//------------ Setup Filesystem
		v, err := NewFs(tmp, fooFile, false)
		myT.FatalfIfErr(err, "Failed to create virtual function")

		err = v.MkdirP("/foo", 0755, time1)
		myT.FatalfIfErr(err, "failed to create virtual folder /foo")

		err = createFile(v, "/foo/bar", 0655, time2, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo/bar")

		fooV, err := v.FsFrom("/foo/bar")
		myT.FatalfIfErr(err, "failed to create virtual filesystem /foo/bar")
		fooV.Error(fmt.Errorf("yikes! somthing whent wrong"))

		err = v.Symlink("/foo/bar", "/foo/bar-symlink", 0777, time3)
		myT.FatalfIfErr(err, "failed to create symlink /foo/bar-symlink")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo/bar", 0655, time2, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/foo/bar-symlink", 0777 | fs.ModeSymlink, time3, "", "symlink/symlink", "/foo/bar", emptyTags},
		}
		myT.AssertFiles(expected, v, "after adding files in virtual file system")
		myT.AssertTmpDirFileCount(2, tmp, "after adding files in virtual file system")
		myT.AssertErr(v.ProcessError(), ErrInFilesystem, "after setting error")
		myT.Assert(v.ProcessWarning() == nil, "should not have warning if not set")

		//------------ Close
		err = v.Close()
		myT.FatalfIfErr(err, "error closing file /foo1/foo2/foo3/bar")
		myT.AssertTmpDirFileCount(3, tmp, "after closing virtual file system")

		//------------ Make sure DB file exists
		_, err = os.Stat(filepath.Join(tmp, "fin.db"))
		myT.AssertEqual(err, nil, "expected db file to exist after closing")

		//------------ Expect error if try to make changes
		err = v.MkdirP("/foo/should-fail-folder", 0755, time1)
		myT.AssertEqual(err, ErrClosed, "expected error creating virtual folder /foo/should-fail-folder after closing")

		_, err = v.Create("/foo/should-fail-file", 0655, time1)
		myT.AssertEqual(err, ErrClosed, "expected error creating virtual file /foo/should-fail-file after closing")

		// Create a symlink
		err = v.Symlink("/foo/bar", "/foo/another-symlink", 0700, time1)
		myT.AssertEqual(err, ErrClosed, "expected error creating symlink /foo/another-symlink after closing")

		//------------ Load folder and make sure it works
		newV, err := NewFsFromDb(tmp)
		myT.FatalfIfErr(err, "failed to create filesystem from dir")
		myT.AssertTmpDirFileCount(2, tmp, "after loading fs should delete manifest")

		err = newV.MkdirP("/foo/new-folder", 0155, time1)
		myT.FatalfIfErr(err, "failed to create virtual folder /foo/new-folder")

		err = createFile(newV, "/foo/duplicate", 0200, time2, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo/new-file")

		// Create a file
		err = createFile(newV, "/foo/new-file", 0655, time3, "Hello, Foo!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo/new-file")

		newExpected := append(
			expected,
			fileinfoTest{"/foo/duplicate", 0200, time2, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			fileinfoTest{"/foo/new-file", 0655, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
			fileinfoTest{"/foo/new-folder", 0155 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
		)
		myT.AssertFiles(newExpected, newV, "after creating files in virtual from")
		myT.AssertTmpDirFileCount(3, tmp, "after creating in virtual from")
		myT.AssertErr(newV.ProcessError(), ErrInFilesystem, "should have error when loading")
		myT.Assert(newV.ProcessWarning() == nil, "should NOT have warning after loading if wasnt set")
	})
}

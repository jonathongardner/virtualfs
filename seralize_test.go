package virtualfs

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
)

func TestVirtualClose(t *testing.T) {
	myT := NewMyT("Test Virtual Close", t)
	myT.TmpFile(func(tmp string) {
		//------------ Setup Filesystem
		file, err := os.Open(fooFile)
		myT.FatalfIfErr(err, "failed to open foo file")

		fileinfo, err := file.Stat()
		myT.FatalfIfErr(err, "failed to stat foo file")

		v, err := NewFsFromFileInfo(tmp, fileinfo, false)
		myT.FatalfIfErr(err, "Failed to create virtual function")

		v.CopyToFs(file)

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
		myT.AssertArchiveSize(fooSize+13, tmp, "after adding files in virtual file system")
		myT.Assert(v.ProcessError() == nil, "should NOT have error if not set")
		myT.AssertErr(v.ProcessWarning(), ErrInFilesystem, "after setting warning")

		//------------ Close
		err = v.Close()
		myT.FatalfIfErr(err, "error closing virtual file")
		// might be fragile
		myT.AssertArchiveSize(fooSize+13+1652, tmp, "after closing virtual file system")

		//------------ Expect error if try to make changes after closes
		err = v.MkdirP("/foo/should-fail-folder", 0755, time1)
		myT.AssertEqual(err, ErrClosed, "expected error creating virtual folder /foo/should-fail-folder after closing")

		_, err = v.CopyTo("/foo/should-fail-file", 0655, time1, strings.NewReader("should fial"))
		myT.AssertEqual(err, ErrClosed, "expected error creating virtual file /foo/should-fail-file after closing")

		// Create a symlink
		err = v.Symlink("/foo/bar", "/foo/another-symlink", 0700, time1)
		myT.AssertEqual(err, ErrClosed, "expected error creating symlink /foo/another-symlink after closing")

		//------------ Load folder
		newV, err := NewFsFromDb(tmp, false)
		myT.FatalfIfErr(err, "failed to open filesystem")
		myT.AssertFiles(expected, newV, "after loading files in virtual from")
		myT.AssertArchiveSize(fooSize+13, tmp, "after loading fs nothing should change")

		//------------ Make sure can add files
		err = newV.MkdirP("/foo/new-folder", 0155, time1)
		myT.FatalfIfErr(err, "failed to create virtual folder /foo/new-folder")

		err = createFile(newV, "/foo/duplicate", 0200, time2, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo/new-file")

		err = createFile(newV, "/foo/new-file", 0655, time3, "Hello, Foo!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo/new-file")

		newExpected := append(
			expected,
			fileinfoTest{"/foo/duplicate", 0200, time2, helloWorldSha512, "text/plain; charset=utf-8", "", map[any]any{"foo2": "bar2", "processed": true}},
			fileinfoTest{"/foo/new-file", 0655, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
			fileinfoTest{"/foo/new-folder", 0155 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
		)
		myT.AssertFiles(newExpected, newV, "after creating files in virtual from")
		myT.AssertArchiveSize(fooSize+13+11, tmp, "should remove file ending")
		myT.Assert(newV.ProcessError() == nil, "should NOT have error on load if it wasnt set before save")
		myT.AssertErr(newV.ProcessWarning(), ErrInFilesystem, "should have warning when loading")

		// ----------- Close again
		err = newV.Close()
		myT.FatalfIfErr(err, "error closing virtual file again")
		myT.AssertArchiveSize(fooSize+13+11+3078, tmp, "should rewrite archive, should be bigger then before")

		//------------ Open as read only
		newV, err = NewFsFromDb(tmp, true)
		myT.FatalfIfErr(err, "failed to open readonly filesystem")
		myT.AssertFiles(newExpected, newV, "after loading files in virtual from")
		myT.AssertArchiveSize(fooSize+13+11+3078, tmp, "after loading fs nothing should change")

		//------------ Expect error if try to make changes since readonly
		err = newV.MkdirP("/foo/should-fail-folder", 0755, time1)
		myT.AssertEqual(ErrReadOnly, err, "expected error creating virtual folder /foo/should-fail-folder for readonly")

		_, err = newV.CopyTo("/foo/should-fail-file", 0655, time1, strings.NewReader("should fial"))
		myT.AssertEqual(err, ErrReadOnly, "expected error creating virtual file /foo/should-fail-file for readonly")

		// Create a symlink
		err = newV.Symlink("/foo/bar", "/foo/another-symlink", 0700, time1)
		myT.AssertEqual(err, ErrReadOnly, "expected error creating symlink /foo/another-symlink for readonly")

		// ----------- Close again again
		err = newV.Close()
		myT.FatalfIfErr(err, "error closing readonly virtual file again")
		myT.AssertArchiveSize(fooSize+13+11+3078, tmp, "should rewrite archive, should be bigger then before")
	})
}

func TestVirtualCloseWithErr(t *testing.T) {
	myT := NewMyT("Test Virtual Close", t)
	myT.TmpFile(func(tmp string) {
		//------------ Setup Filesystem
		file, err := os.Open(fooFile)
		myT.FatalfIfErr(err, "failed to open foo file")

		fileinfo, err := file.Stat()
		myT.FatalfIfErr(err, "failed to stat foo file")

		v, err := NewFsFromFileInfo(tmp, fileinfo, false)
		myT.FatalfIfErr(err, "Failed to create virtual function")

		v.CopyToFs(file)

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
		myT.AssertArchiveSize(fooSize+13, tmp, "after closing virtual file system")
		myT.AssertErr(v.ProcessError(), ErrInFilesystem, "after setting error")
		myT.Assert(v.ProcessWarning() == nil, "should not have warning if not set")

		//------------ Close
		err = v.Close()
		myT.FatalfIfErr(err, "error closing virtual file")
		myT.AssertArchiveSize(fooSize+13+1592, tmp, "after closing virtual file system")

		//------------ Expect error if try to make changes
		err = v.MkdirP("/foo/should-fail-folder", 0755, time1)
		myT.AssertEqual(err, ErrClosed, "expected error creating virtual folder /foo/should-fail-folder after closing")

		_, err = v.CopyTo("/foo/should-fail-file", 0655, time1, strings.NewReader("should fial"))
		myT.AssertEqual(err, ErrClosed, "expected error creating virtual file /foo/should-fail-file after closing")

		// Create a symlink
		err = v.Symlink("/foo/bar", "/foo/another-symlink", 0700, time1)
		myT.AssertEqual(err, ErrClosed, "expected error creating symlink /foo/another-symlink after closing")

		//------------ Load folder and make sure it works
		newV, err := NewFsFromDb(tmp, false)
		myT.FatalfIfErr(err, "failed to create filesystem from dir")
		myT.AssertArchiveSize(fooSize+13, tmp, "after loading fs should truncate manifest")

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
		myT.AssertArchiveSize(fooSize+13+11, tmp, "after creating in virtual from")
		myT.AssertErr(newV.ProcessError(), ErrInFilesystem, "should have error when loading")
		myT.Assert(newV.ProcessWarning() == nil, "should NOT have warning after loading if wasnt set")
	})
}

package virtualfs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestVirtualClose(t *testing.T) {
	tmpDir(t, func(tmp string) {
		//------------ Setup Filesystem
		v, err := newFooFs(tmp)
		fatalfIfErr(t, err, "Failed to create virtual function")

		v.Root().TagS("foo", "bar")
		v.Root().TagS("baz", 47)

		_, err = v.MkdirP("/foo", 0755, time1)
		fatalfIfErr(t, err, "failed to create virtual folder /foo")

		err = createFile(v, "/foo/bar", 0655, time2, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /foo/bar")

		fooV, err := v.FsFrom("/foo/bar")
		fooV.Root().TagS("foo2", "bar2")
		fooV.Root().TagS("processed", true)
		fatalfIfErr(t, err, "failed to create virtual filesystem /foo/bar")
		fooV.Root().Warning(fmt.Errorf("yikes! somthing kinda whent wrong"))

		_, err = v.Symlink("/foo/bar", "/foo/bar-symlink", 0777, time3)
		fatalfIfErr(t, err, "failed to create symlink /foo/bar-symlink")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", map[any]any{"foo": "bar", "baz": 47}},
			{"/foo", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo/bar", 0655, time2, helloWorldSha512, "text/plain; charset=utf-8", "", map[any]any{"foo2": "bar2", "processed": true}},
			{"/foo/bar-symlink", 0777 | fs.ModeSymlink, time3, "", "symlink/symlink", "/foo/bar", emptyTags},
		}
		assertFiles(t, expected, v, "after adding files in virtual file system")
		assertTmpDirFileCount(t, 2, tmp, "after adding files in virtual file system")
		assert(t, v.ProcessError() == nil, "should NOT have error if not set")
		assertErr(t, v.ProcessWarning(), ErrInFilesystem, "after setting warning")

		//------------ Close
		err = v.Close()
		fatalfIfErr(t, err, "error closing file /foo1/foo2/foo3/bar")
		assertTmpDirFileCount(t, 3, tmp, "after closing virtual file system")

		//------------ Make sure DB file exists
		_, err = os.Stat(filepath.Join(tmp, "fin.db"))
		assertEqual(t, err, nil, "expected db file to exist after closing")

		//------------ Expect error if try to make changes
		_, err = v.MkdirP("/foo/should-fail-folder", 0755, time1)
		assertEqual(t, err, ErrClosed, "expected error creating virtual folder /foo/should-fail-folder after closing")

		_, err = v.Create("/foo/should-fail-file", 0655, time1)
		assertEqual(t, err, ErrClosed, "expected error creating virtual file /foo/should-fail-file after closing")

		// Create a symlink
		_, err = v.Symlink("/foo/bar", "/foo/another-symlink", 0700, time1)
		assertEqual(t, err, ErrClosed, "expected error creating symlink /foo/another-symlink after closing")

		//------------ Load folder and make sure it works
		newV, err := NewFsFromDb(tmp)
		fatalfIfErr(t, err, "failed to create filesystem from dir")
		assertFiles(t, expected, newV, "after loading files in virtual from")
		assertTmpDirFileCount(t, 2, tmp, "after loading fs should delete manifest")

		_, err = newV.MkdirP("/foo/new-folder", 0155, time1)
		fatalfIfErr(t, err, "failed to create virtual folder /foo/new-folder")

		err = createFile(newV, "/foo/duplicate", 0200, time2, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /foo/new-file")

		// Create a file
		err = createFile(newV, "/foo/new-file", 0655, time3, "Hello, Foo!")
		fatalfIfErr(t, err, "failed to create virtual file /foo/new-file")

		newExpected := append(
			expected,
			fileinfoTest{"/foo/duplicate", 0200, time2, helloWorldSha512, "text/plain; charset=utf-8", "", map[any]any{"foo2": "bar2", "processed": true}},
			fileinfoTest{"/foo/new-file", 0655, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
			fileinfoTest{"/foo/new-folder", 0155 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
		)
		assertFiles(t, newExpected, newV, "after creating files in virtual from")
		assertTmpDirFileCount(t, 3, tmp, "after creating in virtual from")
		assert(t, newV.ProcessError() == nil, "should NOT have error on load if it wasnt set before save")
		assertErr(t, newV.ProcessWarning(), ErrInFilesystem, "should have warning when loading")
	})
}

func TestVirtualCloseWithErr(t *testing.T) {
	tmpDir(t, func(tmp string) {
		//------------ Setup Filesystem
		v, err := newFooFs(tmp)
		fatalfIfErr(t, err, "Failed to create virtual function")

		_, err = v.MkdirP("/foo", 0755, time1)
		fatalfIfErr(t, err, "failed to create virtual folder /foo")

		err = createFile(v, "/foo/bar", 0655, time2, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /foo/bar")

		fooV, err := v.FsFrom("/foo/bar")
		fatalfIfErr(t, err, "failed to create virtual filesystem /foo/bar")
		fooV.Root().Error(fmt.Errorf("yikes! somthing whent wrong"))

		_, err = v.Symlink("/foo/bar", "/foo/bar-symlink", 0777, time3)
		fatalfIfErr(t, err, "failed to create symlink /foo/bar-symlink")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", "", emptyTags},
			{"/foo", 0755 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
			{"/foo/bar", 0655, time2, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			{"/foo/bar-symlink", 0777 | fs.ModeSymlink, time3, "", "symlink/symlink", "/foo/bar", emptyTags},
		}
		assertFiles(t, expected, v, "after adding files in virtual file system")
		assertTmpDirFileCount(t, 2, tmp, "after adding files in virtual file system")
		assertErr(t, v.ProcessError(), ErrInFilesystem, "after setting error")
		assert(t, v.ProcessWarning() == nil, "should not have warning if not set")

		//------------ Close
		err = v.Close()
		fatalfIfErr(t, err, "error closing file /foo1/foo2/foo3/bar")
		assertTmpDirFileCount(t, 3, tmp, "after closing virtual file system")

		//------------ Make sure DB file exists
		_, err = os.Stat(filepath.Join(tmp, "fin.db"))
		assertEqual(t, err, nil, "expected db file to exist after closing")

		//------------ Expect error if try to make changes
		_, err = v.MkdirP("/foo/should-fail-folder", 0755, time1)
		assertEqual(t, err, ErrClosed, "expected error creating virtual folder /foo/should-fail-folder after closing")

		_, err = v.Create("/foo/should-fail-file", 0655, time1)
		assertEqual(t, err, ErrClosed, "expected error creating virtual file /foo/should-fail-file after closing")

		// Create a symlink
		_, err = v.Symlink("/foo/bar", "/foo/another-symlink", 0700, time1)
		assertEqual(t, err, ErrClosed, "expected error creating symlink /foo/another-symlink after closing")

		//------------ Load folder and make sure it works
		newV, err := NewFsFromDb(tmp)
		fatalfIfErr(t, err, "failed to create filesystem from dir")
		assertTmpDirFileCount(t, 2, tmp, "after loading fs should delete manifest")

		_, err = newV.MkdirP("/foo/new-folder", 0155, time1)
		fatalfIfErr(t, err, "failed to create virtual folder /foo/new-folder")

		err = createFile(newV, "/foo/duplicate", 0200, time2, "Hello, World!")
		fatalfIfErr(t, err, "failed to create virtual file /foo/new-file")

		// Create a file
		err = createFile(newV, "/foo/new-file", 0655, time3, "Hello, Foo!")
		fatalfIfErr(t, err, "failed to create virtual file /foo/new-file")

		newExpected := append(
			expected,
			fileinfoTest{"/foo/duplicate", 0200, time2, helloWorldSha512, "text/plain; charset=utf-8", "", emptyTags},
			fileinfoTest{"/foo/new-file", 0655, time3, helloFooSha512, "text/plain; charset=utf-8", "", emptyTags},
			fileinfoTest{"/foo/new-folder", 0155 | fs.ModeDir, time1, "", "directory/directory", "", emptyTags},
		)
		assertFiles(t, newExpected, newV, "after creating files in virtual from")
		assertTmpDirFileCount(t, 3, tmp, "after creating in virtual from")
		assertErr(t, newV.ProcessError(), ErrInFilesystem, "should have error when loading")
		assert(t, newV.ProcessWarning() == nil, "should NOT have warning after loading if wasnt set")
	})
}

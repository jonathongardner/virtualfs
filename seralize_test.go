package virtualfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVirtualClose(t *testing.T) {
	myT := NewMyT("Test Virtual Close", t)
	myT.TmpDir(func(tmp string) {
		//------------ Setup Filesystem
		v, err := NewFs(tmp, fooFile)
		myT.FatalfIfErr(err, "Failed to create virtual function")

		err = v.MkdirP("/foo", 0755, time1)
		myT.FatalfIfErr(err, "failed to create virtual folder /foo1/foo2")

		err = createFile(v, "/foo/bar", 0655, time2, "Hello, World!")
		myT.FatalfIfErr(err, "failed to create virtual file /foo1/foo2/foo3/bar")

		err = v.Symlink("/foo/bar", "/foo/bar-symlink", 0777, time3)
		myT.FatalfIfErr(err, "failed to create symlink /foo/bar-symlink")

		expected := []fileinfoTest{
			{"/", fooMod, ignoreTime, fooSha512, "application/octet-stream", ""},
			{"/foo", 0755, time1, "", "directory/directory", ""},
			{"/foo/bar", 0655, time2, helloWorldSha512, "text/plain; charset=utf-8", ""},
			{"/foo/bar-symlink", 0777, time3, "", "symlink/symlink", "/foo/bar"},
		}
		myT.AssertFiles(expected, v, "after adding files in virtual file system")
		myT.AssertTmpDirFileCount(2, tmp, "after adding files in virtual file system")

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
		newV, err := NewFsFromDir(tmp)
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
			fileinfoTest{"/foo/duplicate", 0200, time2, helloWorldSha512, "text/plain; charset=utf-8", ""},
			fileinfoTest{"/foo/new-file", 0655, time3, helloFooSha512, "text/plain; charset=utf-8", ""},
			fileinfoTest{"/foo/new-folder", 0155, time1, "", "directory/directory", ""},
		)
		myT.AssertFiles(newExpected, newV, "after creating files in virtual from")
		myT.AssertTmpDirFileCount(3, tmp, "after creating in virtual from")
	})
}

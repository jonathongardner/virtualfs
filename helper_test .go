package virtualfs

import (
	"fmt"
	"os"
	"testing"
)

func TmpDir(fnc func(tmp string)) error {
	dname, err := os.MkdirTemp("", "virtual-testing")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dname)
	fnc(dname)
	return nil
}

type fileinfoTest struct {
	path   string
	mode   os.FileMode
	sha512 string
	ftype  string
}

type MyT struct {
	message string
	t       *testing.T
}

func NewMyT(message string, t *testing.T) *MyT {
	return &MyT{message, t}
}

func (m *MyT) NewMyT(message string) *MyT {
	return &MyT{message, m.t}
}

// --------------------------Base Asserts------------------
func (m *MyT) FatalfIfErr(err error, format string, args ...interface{}) {
	if err == nil {
		return
	}

	str := fmt.Sprintf(format, args...)
	m.t.Fatalf("%v: %v (error: %v)", m.message, str, err)
}

func (m *MyT) AssertEqual(exp any, act any, format string, args ...interface{}) {
	if exp == act {
		return
	}

	str := fmt.Sprintf(format, args...)
	m.t.Fatalf("%v: %v (expected: %v, actual: %v)", m.message, str, exp, act)
}

func (m *MyT) RefuteEqual(exp any, act any, format string, args ...interface{}) {
	if exp != act {
		return
	}

	str := fmt.Sprintf(format, args...)
	m.t.Fatalf("%v: %v (expected: %v, actual: %v)", m.message, str, exp, act)
}

// --------------------------Base Asserts------------------

// --------------------------Fs file------------------
func (t *MyT) AssertFiles(expectedFileInfo []fileinfoTest, v *Fs, format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	count := 0
	v.Walk("/", func(path string, fi os.FileInfo) error {
		t.RefuteEqual(len(expectedFileInfo), 0, "%v %v not expected", str, path)

		expectedFI := expectedFileInfo[0]
		expectedFileInfo = expectedFileInfo[1:]

		n := fi.Sys().(*node)

		t.AssertEqual(expectedFI.path, path, "%v path doesnt match %v", str, count)
		t.AssertEqual(expectedFI.mode, n.mode, "%v mode doesnt match %v", str, count)
		t.AssertEqual(expectedFI.sha512, n.ref.sha512, "%v sha512 doesnt match %v", str, count)
		t.AssertEqual(expectedFI.ftype, n.ref.typ.Mimetype, "%v filetype doesnt match %v", str, count)

		count++
		return nil
	})
	t.AssertEqual(0, len(expectedFileInfo), "recieved more paths then expected")
}
func (t *MyT) AssertPaths(expectedPaths []string, v *Fs, format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)

	count := 0
	v.Walk("/", func(path string, fi os.FileInfo) error {
		expectedPath := expectedPaths[0]
		expectedPaths = expectedPaths[1:]

		t.AssertEqual(expectedPath, path, "%v path doesnt match %v", str, count)
		count++
		return nil
	})
}

//--------------------------Fs file------------------

// --------------------------Tmp Dir------------------
func (t *MyT) AssertTmpDirFileCount(expCnt int, tmp string, format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)

	d, err := os.ReadDir(tmp)
	t.FatalfIfErr(err, "%v failed to read dir", str)

	t.AssertEqual(expCnt, len(d), "%v, file count doesnt match", str)
}

func (t *MyT) TmpDir(fnc func(tmp string)) {
	dname, err := os.MkdirTemp("", "virtual-testing")
	t.FatalfIfErr(err, "Failed to create tmp dir for testing")
	defer os.RemoveAll(dname)
	fnc(dname)
}

// --------------------------Tmp Dir------------------

package virtualfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const fooFile = "testdata/foo"
const fooSha512 = "0f5623276549769a63c79ca20fc573518685819fe82b39f43a3e7cf709c8baa16524daa95e006e81f7267700a88adee8a6209201be960a10c81c35ff3547e3b7"
const fooMod = 0664

var ignoreTime = time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)

const helloWorldSha512 = "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387"
const helloWorldCompressedSha512 = "8f4f138d9d08f1c9f0d5a30a5703886f368b655926bc7823a110511dd83b2e28cf64d90f2825868c7bb5036bb7687bbe7c69e687ad6cf1c351f5b7c619b7b4b5"
const helloFooSha512 = "9b617e0675ac2ede198cfacddf0b283d378a2cee8e72e551a1ae5400cdb9a46792556187e4d2fdbedece0f0021a6b1f74a6b460b62966ef68025abf75fb7df7a"

var time1 = time.Date(2020, 12, 8, 19, 0, 0, 0, time.UTC)
var time2 = time.Date(2022, 4, 7, 22, 0, 0, 0, time.UTC)
var time3 = time.Date(2022, 3, 17, 15, 0, 0, 0, time.UTC)

type fileinfoTest struct {
	path        string
	mode        os.FileMode
	modTime     time.Time
	sha512      string
	ftype       string
	symlinkPath string
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

func (m *MyT) AssertErr(exp error, act error, format string, args ...interface{}) {
	if errors.Is(exp, act) {
		return
	}

	str := fmt.Sprintf(format, args...)
	m.t.Fatalf("%v: %v (expected: %v, actual: %v)", m.message, str, exp, act)
}

func (m *MyT) Assert(v bool, format string, args ...interface{}) {
	if v {
		return
	}

	str := fmt.Sprintf(format, args...)
	m.t.Fatalf("%v: %v", m.message, str)
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
	v.Walk("/", func(path string, fi *FileInfo) error {
		t.RefuteEqual(len(expectedFileInfo), 0, "%v %v not expected", str, path)

		expectedFI := expectedFileInfo[0]
		expectedFileInfo = expectedFileInfo[1:]

		t.AssertEqual(expectedFI.path, path, "%v path doesnt match %v", str, count)
		t.AssertEqual(expectedFI.mode, fi.mode, "%v mode doesnt match %v", str, count)
		if expectedFI.modTime != ignoreTime {
			t.AssertEqual(expectedFI.modTime, fi.modTime, "%v modeTime doesnt match %v", str, count)
		}
		t.AssertEqual(expectedFI.sha512, fi.ref.sha512, "%v sha512 doesnt match %v", str, count)
		t.AssertEqual(expectedFI.ftype, fi.ref.typ.Mimetype, "%v filetype doesnt match %v", str, count)
		t.AssertEqual(expectedFI.symlinkPath, fi.symlinkPath, "%v symlinkPath doesnt match %v", str, count)

		count++
		return nil
	})
	t.AssertEqual(0, len(expectedFileInfo), "recieved more paths then expected")
}
func (t *MyT) AssertPaths(expectedPaths []string, v *Fs, format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)

	count := 0
	v.Walk("/", func(path string, _fi *FileInfo) error {
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
	fnc(filepath.Join(dname, "forklift"))
}

// --------------------------Tmp Dir------------------

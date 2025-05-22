package virtualfs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testFolder = "testdata/foo-folder"

var testMod os.FileMode
var testTime time.Time

const fooFile = "testdata/foo"
const fooSha512 = "0f5623276549769a63c79ca20fc573518685819fe82b39f43a3e7cf709c8baa16524daa95e006e81f7267700a88adee8a6209201be960a10c81c35ff3547e3b7"

var fooMod os.FileMode
var fooTime time.Time

const barFile = testFolder + "/bar"
const barSha512 = "c971808ecc8c67052f1ccce75ca3ac57c75cad6abc1ce7767f7ca515aac311897478eb126dfa1d94042f3881e6fd09bca779dc274938dcaa828fc08ecec94315"

var barMod os.FileMode

const bazFile = testFolder + "/more/baz"
const bazSha512 = "87784f6947fe864688fef50f29004e00e68f79b9a36113b53b4883ae90e0cdf0d7612dcd95079daed17caf9a2b66b0d2f06a7e1ee0984186ca755121f5216894"

var bazMod os.FileMode

func TestMain(m *testing.M) {
	fileInfo, err := os.Stat(testFolder)
	if err != nil {
		panic(err)
	}
	testMod = fileInfo.Mode()
	testTime = fileInfo.ModTime()

	fileInfo, err = os.Stat(fooFile)
	if err != nil {
		panic(err)
	}
	fooMod = fileInfo.Mode()
	fooTime = fileInfo.ModTime()

	fileInfo, err = os.Stat(barFile)
	if err != nil {
		panic(err)
	}
	barMod = fileInfo.Mode()

	fileInfo, err = os.Stat(bazFile)
	if err != nil {
		panic(err)
	}
	bazMod = fileInfo.Mode()

	// fmt.Println("Starting tests...")

	exitCode := m.Run() // Runs all tests

	// Teardown code
	os.Exit(exitCode)
}

var ignoreTime = time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)

const helloWorldSha512 = "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387"
const helloWorldCompressedSha512 = "8f4f138d9d08f1c9f0d5a30a5703886f368b655926bc7823a110511dd83b2e28cf64d90f2825868c7bb5036bb7687bbe7c69e687ad6cf1c351f5b7c619b7b4b5"
const helloFooSha512 = "9b617e0675ac2ede198cfacddf0b283d378a2cee8e72e551a1ae5400cdb9a46792556187e4d2fdbedece0f0021a6b1f74a6b460b62966ef68025abf75fb7df7a"

var time1 = time.Date(2020, 12, 8, 19, 0, 0, 0, time.UTC)
var time2 = time.Date(2022, 4, 7, 22, 0, 0, 0, time.UTC)
var time3 = time.Date(2022, 3, 17, 15, 0, 0, 0, time.UTC)

var emptyTags = map[any]any{}

func newFooFs(tmp string) (*Fs, error) {
	f, err := os.Open(fooFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open foo file %v", err)
	}
	defer f.Close()
	v, err := NewFs(tmp, "foo", fooMod, fooTime, f)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual function %v", err)
	}
	return v, nil
}

func newTestFolderFs(tmp string) (*Fs, error) {
	v, err := NewFs(tmp, "foo-folder", testMod, testTime, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual function %v", err)
	}
	return v, nil
}

type fileinfoTest struct {
	path        string
	mode        os.FileMode
	modTime     time.Time
	sha512      string
	ftype       string
	symlinkPath string
	tags        map[any]any
}

// --------------------------Base Asserts------------------
func fatalfIfErr(t *testing.T, err error, format string, args ...interface{}) {
	t.Helper()
	if err == nil {
		return
	}

	str1 := fmt.Sprintf("expected nil error, got: %v", err)
	str2 := fmt.Sprintf(format, args...)
	t.Fatal(str1, str2)
}

func assertErr(t *testing.T, exp error, act error, format string, args ...interface{}) {
	t.Helper()
	if errors.Is(exp, act) {
		return
	}

	str1 := fmt.Sprintf("exp: %v, act: %v", exp, act)
	str2 := fmt.Sprintf(format, args...)
	t.Error(str1, str2)
}

func assert(t *testing.T, v bool, format string, args ...interface{}) {
	t.Helper()
	if v {
		return
	}

	t.Errorf(format, args...)
}

func assertEqual(t *testing.T, exp any, act any, format string, args ...interface{}) {
	t.Helper()
	if exp == act {
		return
	}

	str1 := fmt.Sprintf("exp: %v, act: %v", exp, act)
	str2 := fmt.Sprintf(format, args...)
	t.Error(str1, str2)
}

// --------------------------Base Asserts------------------

// --------------------------Fs file------------------
func assertFiles(t *testing.T, expectedFileInfo []fileinfoTest, v *Fs, format string, args ...interface{}) {
	t.Helper()
	str := fmt.Sprintf(format, args...)
	count := 0
	// TODO: push thes to a struct than compare
	actFileInfo := []fileinfoTest{}
	v.Walk("/", func(path string, fi *FileInfo) error {
		fit := fileinfoTest{
			path:        path,
			mode:        fi.mode,
			modTime:     fi.modTime,
			sha512:      fi.ref.sha512,
			ftype:       fi.ref.typ.Mimetype,
			symlinkPath: fi.symlinkPath,
			tags:        map[any]any{},
		}
		fi.ref.tags.Range(func(k, v any) bool {
			// assertEqual(t.t, expectedFI.tags[k], v, "%v tags do not match %v", str, k)
			fit.tags[k] = true
			return true
		})

		actFileInfo = append(actFileInfo, fit)
		return nil
	})
	if len(actFileInfo) != len(expectedFileInfo) {
		t.Fatalf("%v file count doesnt match, exp: %v, act: %v", str, len(expectedFileInfo), len(actFileInfo))
	}

	for i, afi := range actFileInfo {
		expectedFI := expectedFileInfo[i]

		assertEqual(t, expectedFI.path, afi.path, "%v path doesnt match %v", str, count)
		assertEqual(t, expectedFI.mode, afi.mode, "%v mode doesnt match %v (%d)", str, count, afi.mode)
		if expectedFI.modTime != ignoreTime {
			assertEqual(t, expectedFI.modTime, afi.modTime, "%v modeTime doesnt match %v", str, count)
		}
		assertEqual(t, expectedFI.sha512, afi.sha512, "%v sha512 doesnt match %v", str, count)
		assertEqual(t, expectedFI.ftype, afi.ftype, "%v filetype doesnt match %v", str, count)

		assertEqual(t, len(expectedFI.tags), len(afi.tags), "%v maps size dont match %v", str, count)

		assertEqual(t, expectedFI.symlinkPath, afi.symlinkPath, "%v symlinkPath doesnt match %v", str, count)
	}
	if t.Failed() {
		t.FailNow()
	}
}
func assertPaths(t *testing.T, expectedPaths []string, v *Fs, format string, args ...interface{}) {
	t.Helper()

	str := fmt.Sprintf(format, args...)

	count := 0
	v.Walk("/", func(path string, _fi *FileInfo) error {
		expectedPath := expectedPaths[0]
		expectedPaths = expectedPaths[1:]

		assertEqual(t, expectedPath, path, "%v path doesnt match %v", str, count)
		count++
		return nil
	})
}

//--------------------------Fs file------------------

// --------------------------Tmp Dir------------------
func assertTmpDirFileCount(t *testing.T, expCnt int, tmp string, format string, args ...interface{}) {
	t.Helper()

	str := fmt.Sprintf(format, args...)

	d, err := os.ReadDir(tmp)
	fatalfIfErr(t, err, "%v failed to read dir", str)

	assertEqual(t, expCnt, len(d), "%v, file count doesnt match", str)
}

func tmpDir(t *testing.T, fnc func(tmp string)) {
	// t.t.Helper()
	dname, err := os.MkdirTemp("", "virtual-testing")
	fatalfIfErr(t, err, "Failed to create tmp dir for testing")
	defer os.RemoveAll(dname)
	fnc(filepath.Join(dname, "forklift"))
}

// --------------------------Tmp Dir------------------

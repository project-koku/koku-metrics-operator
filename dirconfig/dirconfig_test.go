package dirconfig

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/project-koku/koku-metrics-operator/testutils"
)

var errTest = errors.New("test error")

type MockFileInfo struct {
	name  string
	isDir bool
}

func NewMockFileInfo(name string, isDir bool) MockFileInfo {
	return MockFileInfo{
		name:  name,
		isDir: isDir,
	}
}

func (mfi MockFileInfo) Name() string {
	return mfi.name
}

func (mfi MockFileInfo) Size() int64 {
	return 100
}

func (mfi MockFileInfo) Mode() os.FileMode {
	if mfi.isDir {
		return os.ModeDir
	}
	return os.ModeAppend
}

func (mfi MockFileInfo) ModTime() time.Time {
	return time.Unix(110, 0)
}

func (mfi MockFileInfo) IsDir() bool {
	return mfi.isDir
}

func (mfi MockFileInfo) Sys() interface{} {
	return nil
}

func TestDirString(t *testing.T) {
	tcs := []struct {
		path     string
		expected string
	}{
		{"/tmp/dir/cfg", "/tmp/dir/cfg"},
		{"/etc/data", "/etc/data"},
		{"/root/configs/costmanagement", "/root/configs/costmanagement"},
	}

	for _, tc := range tcs {
		dir := &Directory{Path: tc.path}
		if dir.String() != tc.expected {
			t.Errorf("Expected dir.String() to return %v but got %v", tc.expected, dir.String())
		}
	}
}

var listDirFileMock = func(files []os.FileInfo, err error) DirListFunc {
	return func(path string) ([]os.FileInfo, error) {
		return files, err
	}
}
var removeAllMock = func(err error) RemoveAllFunc {
	return func(path string) error {
		return err
	}
}

var statMock = func(err error) StatFunc {
	return func(path string) (os.FileInfo, error) {
		return nil, err
	}
}

var createDirMock = func(err error) DirCreateFunc {
	return func(path string) error {
		return err
	}
}

func TestDirRemoveContents(t *testing.T) {
	dirPath := "/bla/configs/"
	tcs := []struct {
		listDirFile   DirListFunc
		removeAll     RemoveAllFunc
		expectedError error
	}{
		{
			listDirFileMock(nil, fmt.Errorf("Oh no!")),
			removeAllMock(nil),
			fmt.Errorf("RemoveContents: could not read directory: Oh no!"),
		},
		{
			listDirFileMock([]os.FileInfo{}, nil),
			removeAllMock(nil),
			nil,
		},
		{
			listDirFileMock([]os.FileInfo{NewMockFileInfo("/tmp/dir/cfg", false)}, nil),
			removeAllMock(fmt.Errorf("oops")),
			fmt.Errorf("RemoveContents: could not remove file: oops"),
		},
	}

	for _, tc := range tcs {
		dir := &Directory{
			Path: dirPath,
			DirectoryFileSystem: &DirectoryFileSystem{
				ListDirectory:   tc.listDirFile,
				RemoveAll:       tc.removeAll,
				Stat:            statMock(nil),
				CreateDirectory: createDirMock(nil),
			},
		}
		err := dir.RemoveContents()
		if tc.expectedError != nil || err != nil {
			if (tc.expectedError.Error() != err.Error()) || (tc.expectedError != nil && err == nil) || (tc.expectedError == nil && err != nil) {
				t.Errorf("Expected to return error: %v but got %v", tc.expectedError, err)
			}
		}
	}
}

func TestDirExists(t *testing.T) {
	dirPath := testutils.RandomString(10)
	tcs := []struct {
		stat     StatFunc
		expected bool
	}{
		{statMock(fmt.Errorf("bad file")), false},
		{statMock(nil), true},
		{os.Stat, false},
	}
	for _, tc := range tcs {
		dir := &Directory{
			Path: dirPath,
			DirectoryFileSystem: &DirectoryFileSystem{
				ListDirectory:   listDirFileMock(nil, nil),
				RemoveAll:       removeAllMock(nil),
				Stat:            tc.stat,
				CreateDirectory: createDirMock(nil),
			},
		}
		res := dir.Exists()
		if res != tc.expected {
			t.Errorf("Expected doesDirExist to return %v but got %v", tc.expected, res)
		}
	}
}

func TestDirCreate(t *testing.T) {
	dirPath := "/etc/cost-management"
	tcs := []struct {
		name      string
		createDir DirCreateFunc
		expected  error
	}{
		{"create error", createDirMock(fmt.Errorf("bad dir")), errTest},
		{"no error", createDirMock(nil), nil},
	}

	for _, tc := range tcs {
		dir := &Directory{
			Path: dirPath,
			DirectoryFileSystem: &DirectoryFileSystem{
				ListDirectory:   listDirFileMock(nil, nil),
				RemoveAll:       removeAllMock(nil),
				Stat:            statMock(nil),
				CreateDirectory: tc.createDir,
			},
		}
		err := dir.Create()
		if tc.expected != nil && err == nil {
			t.Errorf("%s expected error but got: %v", tc.name, err)
		}
		if tc.expected == nil && err != nil {
			t.Errorf("%s expected nil error but got: %v", tc.name, err)
		}
	}
}

func TestCheckExistsOrRecreate(t *testing.T) {
	tcs := []struct {
		name      string
		stat      StatFunc
		createDir DirCreateFunc
		expected  error
	}{
		{name: "no errors", stat: statMock(nil), createDir: createDirMock(nil), expected: nil},
		{name: "stat error", stat: statMock(fmt.Errorf("Not available")), createDir: createDirMock(nil), expected: nil},
		{name: "create error", stat: statMock(fmt.Errorf("Not available")), createDir: createDirMock(fmt.Errorf(" :shocked: ")), expected: errTest},
	}
	lgr := testutils.TestLogger{}

	for _, tc := range tcs {
		dir := &Directory{
			Path: "/etc/ocp_cfg",
			DirectoryFileSystem: &DirectoryFileSystem{
				ListDirectory:   listDirFileMock(nil, nil),
				RemoveAll:       removeAllMock(nil),
				Stat:            tc.stat,
				CreateDirectory: tc.createDir,
			},
		}
		err := CheckExistsOrRecreate(lgr, *dir)
		if tc.expected != nil && err == nil {
			t.Errorf("%s expected error but got: %v", tc.name, err)
		}
		if tc.expected == nil && err != nil {
			t.Errorf("%s expected nil error but got: %v", tc.name, err)
		}
	}
}

func TestGetDirectoryConfig(t *testing.T) {
	tcs := []struct {
		name      string
		listDir   DirListFunc
		removeAll RemoveAllFunc
		stat      StatFunc
		createDir DirCreateFunc
		expected  error
	}{
		{
			name:      "create/stat error",
			listDir:   listDirFileMock(nil, nil),
			removeAll: removeAllMock(nil),
			stat:      statMock(fmt.Errorf("does not exists")),
			createDir: createDirMock(fmt.Errorf("u shall not pass")),
			expected:  errTest,
		},
		{
			name:      "no errors",
			listDir:   listDirFileMock(nil, nil),
			removeAll: removeAllMock(nil),
			stat:      statMock(nil),
			createDir: createDirMock(nil),
			expected:  nil,
		},
	}

	for _, tc := range tcs {
		dirCfg := &DirectoryConfig{
			DirectoryFileSystem: &DirectoryFileSystem{
				ListDirectory:   tc.listDir,
				RemoveAll:       tc.removeAll,
				Stat:            tc.stat,
				CreateDirectory: tc.createDir,
			},
		}
		err := dirCfg.GetDirectoryConfig()
		if tc.expected != nil && err == nil {
			t.Errorf("%s expected error but got: %v", tc.name, err)
		}
		if tc.expected == nil && err != nil {
			t.Errorf("%s expected nil error but got: %v", tc.name, err)
		}
	}
}

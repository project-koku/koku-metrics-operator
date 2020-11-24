package dirconfig

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/project-koku/korekuta-operator-go/testlogr"
	"github.com/project-koku/korekuta-operator-go/testutils"
)

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
		createDir DirCreateFunc
		expected  error
	}{
		{createDirMock(fmt.Errorf("bad dir")), fmt.Errorf("Create: /etc/cost-management: bad dir")},
		{createDirMock(nil), nil},
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
		if tc.expected != nil || err != nil {
			if (tc.expected.Error() != err.Error()) || (tc.expected != nil && err == nil) || (tc.expected == nil && err != nil) {
				t.Errorf("expected createDir to return '%v' but got '%v'", tc.expected, err)
			}
		}
	}
}

func TestCheckExistsOrRecreate(t *testing.T) {
	tcs := []struct {
		stat      StatFunc
		createDir DirCreateFunc
		expected  error
	}{
		{stat: statMock(nil), createDir: createDirMock(nil), expected: nil},
		{stat: statMock(fmt.Errorf("Not available")), createDir: createDirMock(nil), expected: nil},
		{stat: statMock(fmt.Errorf("Not available")), createDir: createDirMock(fmt.Errorf(":shocked:")), expected: fmt.Errorf("Create: /etc/ocp_cfg: :shocked:")},
	}
	lgr := testlogr.TestLogger{}

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
		if tc.expected != nil || err != nil {
			if (tc.expected.Error() != err.Error()) || (tc.expected != nil && err == nil) || (tc.expected == nil && err != nil) {
				t.Errorf("Expected CheckExistsOrRecreate to return error: %v but got %v", tc.expected, err)
			}
		}
	}
}

func TestGetDirectoryConfig(t *testing.T) {
	tcs := []struct {
		listDir   DirListFunc
		removeAll RemoveAllFunc
		stat      StatFunc
		createDir DirCreateFunc
		expected  error
	}{
		{
			listDir:   listDirFileMock(nil, nil),
			removeAll: removeAllMock(nil),
			stat:      statMock(fmt.Errorf("does not exists")),
			createDir: createDirMock(fmt.Errorf("u shall not pass")),
			expected:  fmt.Errorf("getDirectoryConfig: Create: /tmp/cost-mgmt-operator-reports/: u shall not pass"),
		},
		{
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
		if tc.expected != nil || err != nil {
			if tc.expected != nil && err == nil || tc.expected == nil && err != nil || tc.expected.Error() != err.Error() {
				t.Errorf("Expected GetDirectoryConfig to return error: %v but got %v", tc.expected, err)
			}
		}
	}
}

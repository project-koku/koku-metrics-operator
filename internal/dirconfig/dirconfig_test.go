package dirconfig

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/project-koku/koku-metrics-operator/internal/testutils"
)

var errTest = errors.New("test error")

type MockDirEntry struct {
	MockFileInfo MockFileInfo
}

func NewMockDirEntry(mfi MockFileInfo) MockDirEntry {
	return MockDirEntry{MockFileInfo: mfi}
}

func (mde MockDirEntry) Name() string               { return mde.MockFileInfo.name }
func (mde MockDirEntry) IsDir() bool                { return mde.MockFileInfo.isDir }
func (mde MockDirEntry) Info() (os.FileInfo, error) { return mde.MockFileInfo, nil }
func (mde MockDirEntry) Type() os.FileMode {
	if mde.MockFileInfo.isDir {
		return os.ModeDir
	}
	return os.ModeAppend
}

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

func (mfi MockFileInfo) Name() string       { return mfi.name }
func (mfi MockFileInfo) Size() int64        { return 100 }
func (mfi MockFileInfo) ModTime() time.Time { return time.Unix(110, 0) }
func (mfi MockFileInfo) IsDir() bool        { return mfi.isDir }
func (mfi MockFileInfo) Sys() any           { return nil }
func (mfi MockFileInfo) Mode() os.FileMode {
	if mfi.isDir {
		return os.ModeDir
	}
	return os.ModeAppend
}

func TestMain(m *testing.M) {
	logf.SetLogger(testutils.ZapLogger(true))
	code := m.Run()
	os.Exit(code)
}

func TestGetFiles(t *testing.T) {
	if err := os.Mkdir("empty-dir", 0644); err != nil {
		t.Fatalf("failed to create empty-dir: %v", err)
	}
	defer os.RemoveAll("empty-dir")
	getFilesTests := []struct {
		name string
		path string
		want []string
		err  error
	}{
		{
			name: "path exists with files",
			path: "./test_files",
			want: []string{"test_file"},
			err:  nil,
		},
		{
			name: "path exists with no files",
			path: "./empty-dir",
			want: []string{},
			err:  nil,
		},
		{
			name: "path does not exist",
			path: "./not_real",
			want: nil,
			err:  errTest,
		},
	}
	for _, tt := range getFilesTests {
		dir := &Directory{Path: tt.path}
		got, err := dir.GetFiles()
		if err == nil && tt.err != nil {
			t.Errorf("%s expected error got: %v", tt.name, err)
		}
		if err != nil && tt.err == nil {
			t.Errorf("%s expected nil error got: %v", tt.name, err)
		}
		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("%s got %+v want %+v", tt.name, got, tt.want)
		}
	}
}

func TestGetFilesFullPath(t *testing.T) {
	if err := os.Mkdir("empty-dir", 0644); err != nil {
		t.Fatalf("failed to create empty-dir: %v", err)
	}
	defer os.RemoveAll("empty-dir")
	getFilesTests := []struct {
		name string
		path string
		want []string
		err  error
	}{
		{
			name: "path exists with files",
			path: "./test_files",
			want: []string{filepath.Join("test_files", "test_file")},
			err:  nil,
		},
		{
			name: "path exists with no files",
			path: "./empty-dir",
			want: []string{},
			err:  nil,
		},
		{
			name: "path does not exist",
			path: "./not_real",
			want: nil,
			err:  errTest,
		},
	}
	for _, tt := range getFilesTests {
		dir := &Directory{Path: tt.path}
		got, err := dir.GetFilesFullPath()
		if err == nil && tt.err != nil {
			t.Errorf("%s expected error got: %v", tt.name, err)
		}
		if err != nil && tt.err == nil {
			t.Errorf("%s expected nil error got: %v", tt.name, err)
		}
		if !reflect.DeepEqual(tt.want, got) {
			t.Errorf("%s got %+v want %+v", tt.name, got, tt.want)
		}
	}
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

var listDirFileMock = func(files []fs.DirEntry, err error) DirListFunc {
	return func(path string) ([]fs.DirEntry, error) {
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
			listDirFileMock([]fs.DirEntry{}, nil),
			removeAllMock(nil),
			nil,
		},
		{
			listDirFileMock([]fs.DirEntry{NewMockDirEntry(NewMockFileInfo("/tmp/dir/cfg", false))}, nil),
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
		err := CheckExistsOrRecreate(*dir)
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

func TestCheckConfig(t *testing.T) {
	basePath := "./test_files/config_test"
	tts := []struct {
		name     string
		dirs     map[string]string
		expected bool
	}{
		{
			name:     "parent does not exist",
			dirs:     map[string]string{},
			expected: false,
		},
		{
			name: "reports & staging & upload missing",
			dirs: map[string]string{
				"parent": basePath,
			},
			expected: false,
		},
		{
			name: "staging & upload missing",
			dirs: map[string]string{
				"parent":  basePath,
				"reports": "reports",
			},
			expected: false,
		},
		{
			name: "reports & upload missing",
			dirs: map[string]string{
				"parent":  basePath,
				"staging": "staging",
			},
			expected: false,
		},
		{
			name: "reports & staging missing",
			dirs: map[string]string{
				"parent": basePath,
				"upload": "upload",
			},
			expected: false,
		},
		{
			name: "upload missing",
			dirs: map[string]string{
				"parent":  basePath,
				"reports": "reports",
				"staging": "staging",
			},
			expected: false,
		},
		{
			name: "staging missing",
			dirs: map[string]string{
				"parent":  basePath,
				"reports": "reports",
				"upload":  "upload",
			},
			expected: false,
		},
		{
			name: "reports missing",
			dirs: map[string]string{
				"parent":  basePath,
				"staging": "staging",
				"upload":  "upload",
			},
			expected: false,
		},
		{
			name: "all dirs exist",
			dirs: map[string]string{
				"parent":  basePath,
				"reports": "reports",
				"staging": "staging",
				"upload":  "upload",
			},
			expected: true,
		},
	}
	for _, tt := range tts {
		t.Run(tt.name, func(t *testing.T) {
			defer os.RemoveAll(basePath)
			testDirCfg := &DirectoryConfig{}
			for name, path := range tt.dirs {
				switch name {
				case "parent":
					testDirCfg.Parent = Directory{Path: path}
					if err := testDirCfg.Parent.Create(); err != nil {
						t.Fatalf("%s: failed to create test dir: %v", tt.name, err)
					}
				case "reports":
					testDirCfg.Reports = Directory{Path: filepath.Join(basePath, path)}
					if err := testDirCfg.Reports.Create(); err != nil {
						t.Fatalf("%s: failed to create test dir: %v", tt.name, err)
					}
				case "staging":
					testDirCfg.Staging = Directory{Path: filepath.Join(basePath, path)}
					if err := testDirCfg.Staging.Create(); err != nil {
						t.Fatalf("%s: failed to create test dir: %v", tt.name, err)
					}
				case "upload":
					testDirCfg.Upload = Directory{Path: filepath.Join(basePath, path)}
					if err := testDirCfg.Upload.Create(); err != nil {
						t.Fatalf("%s: failed to create test dir: %v", tt.name, err)
					}
				default:
					t.Fatalf("%s unknown directory: %s", tt.name, name)
				}
			}
			got := testDirCfg.CheckConfig()
			if got != tt.expected {
				t.Errorf("%s expected %t, got %t", tt.name, tt.expected, got)
			}
		})
	}
}

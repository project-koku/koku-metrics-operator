package collector

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/project-koku/koku-metrics-operator/internal/strset"
)

var (
	errCtxTimeout = errors.New("context timeout")
	errTest       = errors.New("test error")
)

type badReader struct{}

func (badReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("Read error")
}

type fakeCSVstruct struct{}

func (f fakeCSVstruct) csvHeader() []string { return []string{"fake-header", "fake-header2"} }
func (f fakeCSVstruct) csvRow() []string    { return []string{"fake-row", "fake-row2"} }
func (f fakeCSVstruct) string() string      { return strings.Join(f.csvRow(), ",") }

type fakeFile struct {
	file      *os.File
	created   bool
	createErr error
}

func (f *fakeFile) getOrCreateFile() (*os.File, bool, error) {
	return f.file, f.created, f.createErr
}
func (f *fakeFile) getName() string {
	return "this is a fake file"
}

type fakeData struct {
	writeErr error
	prefix   string
}

func (f *fakeData) getPrefix() string {
	return f.prefix
}

func (f *fakeData) writeToFile(w io.Writer, s *strset.Set, b bool) error {
	return f.writeErr
}

func getTempFile(t *testing.T, mode os.FileMode, dir string) *os.File {
	tempFile, err := os.CreateTemp(dir, "temp-file-")
	if err != nil {
		t.Fatalf("Failed to create temp file.")
	}
	if err := os.Chmod(tempFile.Name(), mode); err != nil {
		t.Fatalf("Failed to change permissions of temp file.")
	}
	return tempFile
}

func getTempDir(t *testing.T, mode os.FileMode, dir, pattern string) string {
	tempDir, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		t.Fatalf("Failed to create temp folder.")
	}
	if err := os.Chmod(tempDir, mode); err != nil {
		t.Fatalf("Failed to change permissions of temp file.")
	}
	return tempDir
}

func TestWriteReport(t *testing.T) {
	tempDir := getTempDir(t, os.ModePerm, "./test_files", "test-dir-*")
	tempBadFile := getTempFile(t, 0777, tempDir)
	tempBadFile.Close()
	defer os.RemoveAll(tempDir)

	writeReportTests := []struct {
		name   string
		report *report
		want   error
	}{
		{
			name: "successful write",
			report: &report{
				file: &fakeFile{
					file:      getTempFile(t, 0777, tempDir),
					created:   false,
					createErr: nil,
				},
				data: &fakeData{writeErr: nil},
			},
			want: nil,
		},
		{
			name: "failed to create file",
			report: &report{
				file: &fakeFile{
					file:      nil,
					created:   false,
					createErr: errTest,
				},
				data: &fakeData{writeErr: nil},
			},
			want: errTest,
		},
		{
			name: "failed to read file",
			report: &report{
				file: &fakeFile{
					file:      tempBadFile,
					created:   false,
					createErr: nil,
				},
				data: &fakeData{writeErr: nil},
			},
			want: errTest,
		},
		{
			name: "failed to write to file",
			report: &report{
				file: &fakeFile{
					file:      getTempFile(t, 0777, tempDir),
					created:   false,
					createErr: nil,
				},
				data: &fakeData{writeErr: errTest},
			},
			want: errTest,
		},
	}
	for _, tt := range writeReportTests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.report.writeReport()
			if tt.want != nil && got == nil {
				t.Errorf("%s got %v want error", tt.name, got)
			}
			if tt.want == nil && got != nil {
				t.Errorf("%s got %v want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestGetOrCreateFile(t *testing.T) {
	tempDir := getTempDir(t, os.ModePerm, "./test_files", "test-dir-*")
	defer os.RemoveAll(tempDir)

	tempDirNoPerm := getTempDir(t, os.ModeDir, "./test_files", "test-dir-*")
	defer os.RemoveAll(tempDirNoPerm)

	tempFileNoPerm := getTempFile(t, 0000, tempDir)

	getOrCreateTests := []struct {
		name          string
		report        *file
		wantedFile    string
		wantedCreated bool
		err           error
	}{
		{
			name: "get existing file",
			report: &file{
				name: "single-line.csv",
				path: "./test_files",
			},
			wantedFile:    "test_files/single-line.csv",
			wantedCreated: false,
			err:           nil,
		},
		{
			name: "create new file",
			report: &file{
				name: "file-to-create.csv",
				path: tempDir,
			},
			wantedFile:    filepath.Join(tempDir, "file-to-create.csv"),
			wantedCreated: true,
			err:           nil,
		},
		{
			name: "create directory in directory with no permissions",
			report: &file{
				name: "file-to-create.csv",
				path: filepath.Join(tempDir, "new-dir"),
			},
			wantedFile:    filepath.Join(tempDir, "new-dir", "file-to-create.csv"),
			wantedCreated: true,
			err:           nil,
		},
		{
			name: "create file in directory with no permissions",
			report: &file{
				name: "file-to-create.csv",
				path: tempDirNoPerm,
			},
			wantedFile:    "",
			wantedCreated: false,
			err:           errTest,
		},
		{
			name: "existing file with no permissions",
			report: &file{
				name: tempFileNoPerm.Name(),
				path: tempDir,
			},
			wantedFile:    "",
			wantedCreated: true,
			err:           errTest,
		},
	}
	for _, tt := range getOrCreateTests {
		t.Run(tt.name, func(t *testing.T) {
			file, created, err := tt.report.getOrCreateFile()
			if tt.err == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if tt.err != nil && err == nil {
				t.Errorf("%s did not error as expected", tt.name)
			}
			if tt.wantedFile != "" && file.Name() != tt.wantedFile {
				t.Errorf("%s got file name %s wanted %s", tt.name, file.Name(), tt.wantedFile)
			}
			if tt.wantedCreated != created {
				t.Errorf("%s got %t wanted %t", tt.name, created, tt.wantedCreated)
			}
		})
	}
}

func TestWriteToFile(t *testing.T) {
	fakeQueryData := mappedCSVStruct{}
	fakeQueryData["fake"] = fakeCSVstruct{}

	builder := &strings.Builder{}

	writeToFileTests := []struct {
		name     string
		report   *data
		set      *strset.Set
		writer   io.Writer
		created  bool
		expected string
		err      error
	}{
		{
			name:     "write header to writer",
			report:   &data{headers: []string{"header1"}},
			set:      strset.NewSet(),
			writer:   builder,
			created:  true,
			expected: "header1\n",
			err:      nil,
		},
		{
			name: "write headers and fake data to writer",
			report: &data{
				headers:   fakeCSVstruct{}.csvHeader(),
				queryData: fakeQueryData,
			},
			set:      strset.NewSet(),
			writer:   builder,
			created:  true,
			expected: "header1\nfake-header,fake-header2\nfake-row,fake-row2\n",
			err:      nil,
		},
		{
			name: "write fake data to writer",
			report: &data{
				headers:   fakeCSVstruct{}.csvHeader(),
				queryData: fakeQueryData,
			},
			set:      strset.NewSet(),
			writer:   builder,
			created:  false,
			expected: "header1\nfake-header,fake-header2\nfake-row,fake-row2\nfake-row,fake-row2\n",
			err:      nil,
		},
	}
	for _, tt := range writeToFileTests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.report.writeToFile(tt.writer, tt.set, tt.created)
			if err != nil && tt.err == nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if tt.err != nil && err == nil {
				t.Errorf("%s got %v error", tt.name, err)
			}
			if tt.expected != "" && builder.String() != tt.expected {
				t.Errorf("%s got %s want %s", tt.name, builder, tt.expected)
			}
		})
	}
}

func TestReadCSV(t *testing.T) {
	testHeaders := "report_period_start,report_period_end,interval_start,interval_end,node\n"
	testRow := "2020-11-01 00:00:00 +0000 UTC,2020-12-01 00:00:00 +0000 UTC,2020-11-06 15:00:00 +0000 UTC,2020-11-06 15:59:59 +0000 UTC,ip-10-0-208-111.us-east-2.compute.internal,openshift-machine-config-operator"
	testSet := strset.NewSet()
	testSet.Add(testRow)

	readCSVTests := []struct {
		name       string
		handle     io.Reader
		set        *strset.Set
		prefix     string
		wantedKeys []string
		err        error
	}{
		{
			name:       "read file success without matching prefix",
			handle:     strings.NewReader("These are the headers\nThis is a test string"),
			set:        strset.NewSet(),
			prefix:     "Not matching",
			wantedKeys: []string{},
		},
		{
			name:       "read file success with matching prefix",
			handle:     strings.NewReader(testHeaders + testRow),
			set:        strset.NewSet(),
			prefix:     "2020-11-01 00:00:00 +0000 UTC,2020-12-01 00:00:00 +0000 UTC,2020-11-06 15:00:00 +0000 UTC,2020-11-06 15:59:59 +0000 UTC",
			wantedKeys: []string{testRow},
		},
		{
			name:       "scanner error",
			handle:     badReader{},
			set:        strset.NewSet(),
			prefix:     "",
			wantedKeys: []string{},
			err:        errTest,
		},
	}
	for _, tt := range readCSVTests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readCSV(tt.handle, tt.set, tt.prefix)
			if tt.err == nil && err != nil {
				t.Errorf("%s got unexpected error: %v", tt.name, err)
			}
			if tt.err != nil && err == nil {
				t.Errorf("%s got `%v` error, wanted error", tt.name, err)
			}
			if got.Len() != len(tt.wantedKeys) {
				t.Errorf("%s lengths not equal, got %v, want %v", tt.name, got, len(tt.wantedKeys))
			}
			for _, key := range tt.wantedKeys {
				if !got.Contains(key) {
					t.Errorf("%s does not contain wanted key: %s", tt.name, key)
				}
			}
		})
	}
}

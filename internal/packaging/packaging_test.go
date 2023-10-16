//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package packaging

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/internal/dirconfig"
	"github.com/project-koku/koku-metrics-operator/internal/testutils"
)

var testingDir string
var dirCfg *dirconfig.DirectoryConfig = new(dirconfig.DirectoryConfig)
var cr = &metricscfgv1beta1.MetricsConfig{}
var testPackager = FilePackager{
	DirCfg:      dirCfg,
	FilesAction: MoveFiles,
}
var errTest = errors.New("test error")

type fakeManifest struct{}

func (m fakeManifest) MarshalJSON() ([]byte, error) {
	return nil, errors.New("This is a marshaling error")
}

// setup a testDirConfig helper for tests
type testDirConfig struct {
	directory string
	files     []os.FileInfo
}
type testDirMap struct {
	large           testDirConfig
	small           testDirConfig
	manifest        testDirConfig
	manifestBad     testDirConfig
	moving          testDirConfig
	tar             testDirConfig
	restricted      testDirConfig
	empty           testDirConfig
	restrictedEmpty testDirConfig
}

var testDirs testDirMap

func Copy(mode os.FileMode, src, dst string) (os.FileInfo, error) {
	err := copyFile(src, dst)
	if err != nil {
		return nil, err
	}
	out, err := os.Open(dst)
	if err != nil {
		return nil, err
	}

	if err := os.Chmod(out.Name(), mode); err != nil {
		return nil, err
	}

	return out.Stat()
}

func getTempFile(t *testing.T, mode os.FileMode, dir string) *os.File {
	tempFile, err := os.CreateTemp(".", "garbage-file")
	if err != nil {
		t.Errorf("Failed to create temp file.")
	}
	setPerm(t, mode, tempFile.Name())
	return tempFile
}

func getTempDir(t *testing.T, mode os.FileMode, dir, pattern string) string {
	tempDir, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		t.Fatalf("Failed to create temp folder.")
	}
	setPerm(t, mode, tempDir)
	return tempDir
}

func setPerm(t *testing.T, mode os.FileMode, dir string) {
	if err := os.Chmod(dir, mode); err != nil {
		t.Fatalf("Failed to change permissions of temp folder.")
	}
}

func genDirCfg(t *testing.T, dirName string) *dirconfig.DirectoryConfig {
	dirCfg := dirconfig.DirectoryConfig{
		Parent:  dirconfig.Directory{Path: dirName},
		Upload:  dirconfig.Directory{Path: filepath.Join(dirName, "upload")},
		Staging: dirconfig.Directory{Path: filepath.Join(dirName, "staging")},
		Reports: dirconfig.Directory{Path: filepath.Join(dirName, "data")},
	}
	if err := dirconfig.CheckExistsOrRecreate(
		dirCfg.Upload,
		dirCfg.Staging,
		dirCfg.Reports,
	); err != nil {
		t.Logf("failed to create dirCfg: %v", err)
	}
	return &dirCfg
}

func setup() error {
	type dirInfo struct {
		dirName  string
		files    []string
		dirMode  os.FileMode
		fileMode os.FileMode
	}
	testFiles := []string{"ocp_node_label.csv", "nonCSV.txt", "ocp_pod_label.csv", "ros-openshift.csv"}
	dirInfoList := []dirInfo{
		{
			dirName:  "large",
			files:    testFiles,
			dirMode:  0777,
			fileMode: 0777,
		},
		{
			dirName:  "small",
			files:    testFiles,
			dirMode:  0777,
			fileMode: 0777,
		},
		{
			dirName:  "manifest",
			files:    testFiles,
			dirMode:  0777,
			fileMode: 0777,
		},
		{
			dirName:  "manifestBad",
			files:    []string{"ocp_pod_missing_header.csv", "ocp_pod_missing_end.csv", "ocp_pod_missing_start.csv", "nonCSV.txt"},
			dirMode:  0777,
			fileMode: 0777,
		},
		{
			dirName:  "moving",
			files:    testFiles,
			dirMode:  0777,
			fileMode: 0777,
		},
		{
			dirName:  "tar",
			files:    testFiles,
			dirMode:  0777,
			fileMode: 0777,
		},
		{
			dirName:  "restricted",
			files:    []string{"bad-csv.csv"},
			dirMode:  0777,
			fileMode: 0000,
		},
		{
			dirName: "empty",
			files:   []string{},
			dirMode: 0777,
		},
		{
			dirName: "restrictedEmpty",
			files:   []string{},
			dirMode: 0000,
		},
	}
	// setup the initial testing directory
	log.Info("Setting up for packaging tests")
	testingUUID := uuid.New().String()
	testingDir = filepath.Join("test_files/", testingUUID)
	if _, err := os.Stat(testingDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testingDir, os.ModePerm); err != nil {
			return fmt.Errorf("could not create %s directory: %v", testingDir, err)
		}
	}
	for _, directory := range dirInfoList {
		reportPath := filepath.Join(testingDir, directory.dirName)
		reportDataPath := filepath.Join(reportPath, "data")
		if _, err := os.Stat(reportPath); os.IsNotExist(err) {
			if err := os.Mkdir(reportPath, directory.dirMode); err != nil {
				return fmt.Errorf("could not create %s directory: %v", reportPath, err)
			}
			if directory.dirName != "empty" && directory.dirName != "restrictedEmpty" {
				if err := os.Mkdir(reportDataPath, directory.dirMode); err != nil {
					return fmt.Errorf("could not create %s directory: %v", reportDataPath, err)
				}
			}
			fileList := []os.FileInfo{}
			for _, reportFile := range directory.files {
				fileInfo, err := Copy(directory.fileMode, filepath.Join("test_files/", reportFile), filepath.Join(reportDataPath, reportFile))
				if err != nil {
					return fmt.Errorf("could not copy %s file: %v", reportFile, err)
				}
				fileList = append(fileList, fileInfo)
			}

			tmpDirMap := testDirConfig{
				directory: reportPath,
				files:     fileList,
			}
			switch directory.dirName {
			case "large":
				testDirs.large = tmpDirMap
			case "small":
				testDirs.small = tmpDirMap
			case "manifest":
				testDirs.manifest = tmpDirMap
			case "manifestBad":
				testDirs.manifestBad = tmpDirMap
			case "moving":
				testDirs.moving = tmpDirMap
			case "empty":
				testDirs.empty = tmpDirMap
			case "restricted":
				testDirs.restricted = tmpDirMap
			case "restrictedEmpty":
				testDirs.restrictedEmpty = tmpDirMap
			case "tar":
				testDirs.tar = tmpDirMap
			default:
				return fmt.Errorf("unknown directory.dirName")
			}
		}
	}
	return nil
}

func shutdown() {
	log.Info("tearing down for packaging tests")
	os.RemoveAll(testingDir)
}

func TestMain(m *testing.M) {
	logf.SetLogger(testutils.ZapLogger(true))
	code := 1 // default to failing code
	err := setup()
	if err != nil {
		log.Info("test setup failed: %v", err)
	} else {
		code = m.Run()
	}
	shutdown()
	os.Exit(code)
}

func TestNeedSplit(t *testing.T) {
	// create the needSplitTests
	cr = &metricscfgv1beta1.MetricsConfig{}
	needSplitTests := []struct {
		name     string
		fileList []os.FileInfo
		maxBytes int64
		want     bool
	}{
		{
			name:     "test split required",
			fileList: testDirs.large.files,
			maxBytes: 1 * 1024 * 1024,
			want:     true,
		},
		{
			name:     "test split not required",
			fileList: testDirs.small.files,
			maxBytes: 100 * 1024 * 1024,
			want:     false,
		},
	}
	for _, tt := range needSplitTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			testPackager.maxBytes = tt.maxBytes
			got := testPackager.needSplit(tt.fileList)
			if tt.want != got {
				t.Errorf("Outcome for test %s:\nReceived: %v\nExpected: %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestBuildLocalCSVFileList(t *testing.T) {
	// create the buildLocalCSVFileList tests
	cr = &metricscfgv1beta1.MetricsConfig{}
	buildLocalCSVFileListTests := []struct {
		name     string
		dirName  string
		fileList []os.FileInfo
	}{
		{
			name:     "test regular dir",
			dirName:  filepath.Join(testDirs.large.directory, "data"),
			fileList: testDirs.large.files,
		},
		{
			name:     "test empty dir",
			dirName:  filepath.Join(testDirs.empty.directory, "data"),
			fileList: testDirs.empty.files,
		},
	}
	for _, tt := range buildLocalCSVFileListTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			got := testPackager.buildLocalCSVFileList(tt.fileList, tt.dirName)
			want := newFileTracker()
			for idx, file := range tt.fileList {
				// generate the expected file list
				if strings.HasSuffix(file.Name(), ".csv") {
					want.allfiles[idx] = filepath.Join(tt.dirName, file.Name())
				}
			}
			if !reflect.DeepEqual(want.allfiles, got.allfiles) {
				t.Errorf("%s expected %v but got %v", tt.name, want.allfiles, got.allfiles)
			}
		})
	}
}

func TestMoveFiles(t *testing.T) {
	// create the moveFiles tests
	cr = &metricscfgv1beta1.MetricsConfig{}
	moveFilesTests := []struct {
		name      string
		dirName   string
		fileUUID  string
		want      []os.FileInfo
		expectErr bool
	}{
		{
			name:      "test moving dir",
			dirName:   testDirs.moving.directory,
			fileUUID:  uuid.New().String(),
			want:      testDirs.moving.files,
			expectErr: false,
		},
		{
			name:      "test empty dir",
			dirName:   testDirs.empty.directory,
			fileUUID:  uuid.New().String(),
			want:      nil,
			expectErr: true,
		},
	}
	for _, tt := range moveFilesTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			testPackager.DirCfg = genDirCfg(t, tt.dirName)
			testPackager.uid = tt.fileUUID
			got, err := testPackager.moveOrCopyFiles(cr)
			if tt.want == nil && got != nil {
				t.Errorf("Expected moved files to be nil")
			} else if tt.want != nil && got == nil {
				t.Errorf("Expected moved files to not be nil")
			} else {
				for _, file := range got {
					// check that the file contains the uuid
					if !strings.Contains(file.Name(), tt.fileUUID) {
						t.Errorf("Expected %s to be in the name but got %s", tt.fileUUID, file.Name())
					}
					// check that the file exists in the staging directory
					if _, err := os.Stat(filepath.Join(filepath.Join(tt.dirName, "staging"), file.Name())); os.IsNotExist(err) {
						t.Errorf("File does not exist in the staging directory")
					}
				}
			}
			if err != nil && !tt.expectErr {
				t.Errorf("%s an unexpected error occurred %v", tt.name, err)
			}
			if err == nil && tt.expectErr {
				t.Errorf("%s expected error but got %v", tt.name, err)
			}
		})
	}
}

func TestPackagingReports(t *testing.T) {
	// create the packagingReports tests
	cr = &metricscfgv1beta1.MetricsConfig{}
	packagingReportTests := []struct {
		name          string
		dirCfg        *dirconfig.DirectoryConfig
		maxReports    int64
		maxSize       int64
		multipleFiles bool
		want          error
		expectFiles   bool
	}{
		{
			name:          "test large dir",
			dirCfg:        genDirCfg(t, testDirs.large.directory),
			maxReports:    10,
			maxSize:       1,
			multipleFiles: true,
			want:          nil,
			expectFiles:   true,
		},
		{
			name:          "test small dir",
			dirCfg:        genDirCfg(t, testDirs.small.directory),
			maxReports:    10,
			maxSize:       100,
			multipleFiles: false,
			want:          nil,
			expectFiles:   true,
		},
		{
			name:          "test empty dir",
			dirCfg:        genDirCfg(t, testDirs.empty.directory),
			maxReports:    10,
			maxSize:       100,
			multipleFiles: false,
			want:          nil,
			expectFiles:   false,
		},
		{
			name:          "restricted",
			dirCfg:        genDirCfg(t, testDirs.restrictedEmpty.directory),
			maxReports:    10,
			maxSize:       100,
			multipleFiles: false,
			want:          errors.New("Restricted"),
			expectFiles:   false,
		},
		{
			name:          "nonexistent",
			dirCfg:        new(dirconfig.DirectoryConfig),
			maxReports:    10,
			maxSize:       100,
			multipleFiles: false,
			want:          errors.New("nonexistent"),
			expectFiles:   false,
		},
	}
	for _, tt := range packagingReportTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			testPackager.DirCfg = tt.dirCfg
			cr.Spec.Packaging.MaxReports = tt.maxReports
			cr.Status.Packaging.MaxSize = &tt.maxSize
			err := testPackager.PackageReports(cr)
			if tt.want != nil && err == nil {
				t.Errorf("%s wanted error got %v", tt.name, err)
			}
			if tt.want == nil && err != nil {
				t.Errorf("%s unexpected err: %v", tt.name, err)
			}
			outFiles, _ := tt.dirCfg.Upload.GetFiles()
			if tt.multipleFiles && len(outFiles) <= 1 || (!tt.multipleFiles && len(outFiles) > 1) {
				t.Errorf("%s expected multpile files: %v, received: %d", tt.name, tt.multipleFiles, len(outFiles))
			}
			if tt.expectFiles && len(outFiles) < 1 {
				t.Errorf("%s expected files to exist", tt.name)
			}
		})
	}
}

func TestGetAndRenderManifest(t *testing.T) {
	// set up the tests to check the manifest contents
	cr = &metricscfgv1beta1.MetricsConfig{}
	getAndRenderManifestTests := []struct {
		name          string
		dirCfg        string
		dirName       string
		fileList      []os.FileInfo
		podReportName string
		expectErr     bool
	}{
		{
			name:          "test regular dir",
			dirCfg:        testDirs.manifest.directory,
			dirName:       filepath.Join(testDirs.manifest.directory, "data"),
			fileList:      testDirs.large.files,
			podReportName: "ocp_pod_label.csv",
			expectErr:     false,
		},
		{
			name:          "test missing header",
			dirCfg:        testDirs.manifestBad.directory,
			dirName:       filepath.Join(testDirs.manifestBad.directory, "data"),
			fileList:      testDirs.large.files,
			podReportName: "ocp_pod_missing_header.csv",
			expectErr:     true,
		},
		{
			name:          "test missing start interval",
			dirCfg:        testDirs.manifestBad.directory,
			dirName:       filepath.Join(testDirs.manifestBad.directory, "data"),
			fileList:      testDirs.large.files,
			podReportName: "ocp_pod_missing_start.csv",
			expectErr:     true,
		},
		{
			name:          "test missing end interval",
			dirCfg:        testDirs.manifestBad.directory,
			dirName:       filepath.Join(testDirs.manifestBad.directory, "data"),
			fileList:      testDirs.large.files,
			podReportName: "ocp_pod_missing_end.csv",
			expectErr:     true,
		},
		{
			name:          "test empty file",
			dirCfg:        testDirs.manifestBad.directory,
			dirName:       filepath.Join(testDirs.manifestBad.directory, "data"),
			fileList:      testDirs.large.files,
			podReportName: "nonCSV.txt",
			expectErr:     true,
		},
		{
			name:          "test empty dir",
			dirCfg:        testDirs.empty.directory,
			dirName:       filepath.Join(testDirs.empty.directory, "staging"),
			fileList:      testDirs.empty.files,
			podReportName: "ocp_pod_label.csv",
			expectErr:     true,
		},
	}
	for _, tt := range getAndRenderManifestTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			testPackager.DirCfg = genDirCfg(t, tt.dirCfg)
			csvFileNames := testPackager.buildLocalCSVFileList(tt.fileList, tt.dirName)
			if err := testPackager.getStartEnd(filepath.Join(testPackager.DirCfg.Reports.Path, tt.podReportName)); err != nil {
				if !tt.expectErr {
					log.Info("This error occurred", "error", err)
					t.Fatal("could not set start/end times")
				}
			}
			testPackager.getManifest(csvFileNames, tt.dirName, cr)
			if err := testPackager.manifest.renderManifest(); err != nil {
				t.Fatal("failed to render manifest")
			}
			// check that the manifest was generated correctly
			if testPackager.manifest.filename != filepath.Join(tt.dirName, "manifest.json") {
				t.Errorf("Manifest was not generated correctly")
			}
			// check that the manifest content is correct
			manifestData, _ := os.ReadFile(testPackager.manifest.filename)
			var foundManifest manifest
			err := json.Unmarshal(manifestData, &foundManifest)
			if err != nil {
				t.Errorf("Error unmarshaling manifest")
			}
			// Define the expected manifest
			var expectedCostFiles []string
			for idx := range csvFileNames.costfiles {
				uploadName := testPackager.uid + "_openshift_usage_report." + strconv.Itoa(idx) + ".csv"
				expectedCostFiles = append(expectedCostFiles, uploadName)
			}
			var expectedRosFiles []string
			for idx := range csvFileNames.rosfiles {
				uploadName := testPackager.uid + "_openshift_usage_report." + strconv.Itoa(idx) + ".csv"
				expectedRosFiles = append(expectedRosFiles, uploadName)
			}
			manifestDate := metav1.Now()

			// getting the start and end time from the ocp_pod_label test csv
			startTime, _ := time.Parse("2006-01-02 15:04:05", strings.Split("2021-01-05 18:00:00", " +")[0])
			endTime, _ := time.Parse("2006-01-02 15:04:05", strings.Split("2021-01-07 18:59:59", " +")[0])
			expectedManifest := manifest{
				UUID:      testPackager.uid,
				ClusterID: cr.Status.ClusterID,
				CRStatus:  cr.Status,
				Version:   cr.Status.OperatorCommit,
				Date:      manifestDate.UTC(),
				Files:     expectedCostFiles,
				ROSFiles:  expectedRosFiles,
				Start:     startTime.UTC(),
				End:       endTime.UTC(),
			}
			// Compare the found manifest to the expected manifest
			errorMsg := "Manifest does not match. Expected %s, recieved %s"
			if foundManifest.UUID != expectedManifest.UUID {
				t.Errorf(errorMsg, expectedManifest.UUID, foundManifest.UUID)
			}
			if foundManifest.ClusterID != expectedManifest.ClusterID {
				t.Errorf(errorMsg, expectedManifest.ClusterID, foundManifest.ClusterID)
			}
			if foundManifest.Version != expectedManifest.Version {
				t.Errorf(errorMsg, expectedManifest.Version, foundManifest.Version)
			}
			if foundManifest.Start != expectedManifest.Start {
				t.Errorf(errorMsg, expectedManifest.Start, foundManifest.Start)
			}
			if foundManifest.End != expectedManifest.End {
				t.Errorf(errorMsg, expectedManifest.End, foundManifest.End)
			}
			if foundManifest.CRStatus.Upload.UploadToggle != expectedManifest.CRStatus.Upload.UploadToggle {
				t.Errorf(errorMsg, expectedManifest.End, foundManifest.End)
			}
			for _, file := range expectedCostFiles {
				found := false
				for _, foundFile := range foundManifest.Files {
					if file == foundFile {
						found = true
					}
				}
				if !found {
					t.Errorf(errorMsg, file, foundManifest.Files)
				}
			}
			if len(foundManifest.Files) != len(expectedCostFiles) {
				t.Errorf("%s manifest filelist length does not match. Expected %d, got %d", tt.name, len(expectedCostFiles), len(foundManifest.Files))
			}
			for _, file := range expectedRosFiles {
				found := false
				for _, foundFile := range foundManifest.ROSFiles {
					if file == foundFile {
						found = true
					}
				}
				if !found {
					t.Errorf(errorMsg, file, foundManifest.ROSFiles)
				}
			}
			if len(foundManifest.ROSFiles) != len(expectedRosFiles) {
				t.Errorf("%s manifest filelist length does not match. Expected %d, got %d", tt.name, len(expectedRosFiles), len(foundManifest.ROSFiles))
			}
		})
	}
}

func TestRenderManifest(t *testing.T) {
	cr = &metricscfgv1beta1.MetricsConfig{}
	tempFile := getTempFile(t, 0644, ".")
	tempFileNoPerm := getTempFile(t, 0000, ".")
	defer os.Remove(tempFile.Name())
	defer os.Remove(tempFileNoPerm.Name())
	renderManifestTests := []struct {
		name  string
		input manifestInfo
		want  string
	}{
		{
			name: "success path",
			input: manifestInfo{
				manifest: manifest{},
				filename: tempFile.Name(),
			},
			want: "",
		},
		{
			name: "file with permission denied",
			input: manifestInfo{
				manifest: manifest{},
				filename: tempFileNoPerm.Name(),
			},
			want: "permission denied",
		},
		{
			name: "json marshalling error",
			input: manifestInfo{
				manifest: fakeManifest{},
				filename: "",
			},
			want: "This is a marshaling error",
		},
	}
	for _, tt := range renderManifestTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.renderManifest()
			if tt.want != "" && !strings.Contains(got.Error(), tt.want) {
				t.Errorf("Outcome for test %s:\nReceived: %s\nExpected: %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestWriteTarball(t *testing.T) {
	// setup the writeTarball tests
	cr = &metricscfgv1beta1.MetricsConfig{}
	writeTarballTests := []struct {
		name         string
		dirName      string
		fileList     []os.FileInfo
		manifestName string
		tarFileName  string
		genCSVs      bool
		expectedErr  bool
	}{
		{
			name:         "test regular dir",
			dirName:      testDirs.tar.directory,
			fileList:     testDirs.tar.files,
			manifestName: "",
			tarFileName:  "cost.tar.gz",
			genCSVs:      true,
			expectedErr:  false,
		},
		{
			name:         "test mismatched files",
			dirName:      testDirs.large.directory,
			fileList:     testDirs.large.files,
			manifestName: "",
			tarFileName:  "cost.tar.gz",
			genCSVs:      true,
			expectedErr:  true,
		},
		{
			name:         "test empty dir",
			dirName:      testDirs.empty.directory,
			fileList:     testDirs.empty.files,
			manifestName: "",
			tarFileName:  "cost.tar.gz",
			genCSVs:      true,
			expectedErr:  false,
		},
		{
			name:         "test nonexistant manifest path",
			dirName:      testDirs.large.directory,
			fileList:     testDirs.large.files,
			manifestName: testPackager.manifest.filename + "nonexistent",
			tarFileName:  "cost.tar.gz",
			genCSVs:      false,
			expectedErr:  true,
		},
		{
			name:         "test bad tarfile path",
			dirName:      testDirs.large.directory,
			fileList:     testDirs.large.files,
			manifestName: "",
			tarFileName:  filepath.Join(uuid.New().String(), "cost-mgmt.tar.gz"),
			genCSVs:      true,
			expectedErr:  true,
		},
	}
	for _, tt := range writeTarballTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			testPackager.DirCfg = genDirCfg(t, tt.dirName)
			stagingDir := testPackager.DirCfg.Reports.Path
			csvFileNames := fileTracker{}
			if tt.genCSVs {
				csvFileNames = testPackager.buildLocalCSVFileList(tt.fileList, stagingDir)
			}
			manifestName := tt.manifestName
			testPackager.getManifest(csvFileNames, stagingDir, cr)
			if err := testPackager.manifest.renderManifest(); err != nil {
				t.Fatal("failed to render manifest")
			}
			if tt.manifestName == "" {
				manifestName = testPackager.manifest.filename
			}
			err := testPackager.writeTarball(tt.tarFileName, manifestName, csvFileNames.allfiles)
			// ensure the tarfile was created if we expect it to be
			if !tt.expectedErr {
				filePath := filepath.Join(testPackager.DirCfg.Upload.Path, tt.tarFileName)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("Tar file was not created")
				}
				// the only testcases that should generate tars are the normal use case
				// and the empty use case
				// if the regular test case, there should be a manifest and 3 csv files
				numFiles := 4
				if strings.Contains(tt.name, "empty") {
					// if the test case is the empty dir, there should only be a manifest
					numFiles = 1
				}
				// check the contents of the tarball
				file, err := os.Open(filePath)
				if err != nil {
					t.Errorf("Can not open tarfile generated by %s", tt.name)
				}
				archive, err := gzip.NewReader(file)
				if err != nil {
					t.Errorf("Can not read tarfile generated by %s", tt.name)
				}
				tr := tar.NewReader(archive)
				var files []string
				for {
					hdr, err := tr.Next()
					if err == io.EOF {
						break
					}

					if err != nil {
						log.Info("%s error: %v", tt.name, err)
						t.Errorf("An error occurred reading the tarfile generated by %s", tt.name)
					}
					files = append(files, hdr.Name)
				}
				if len(files) != numFiles {
					t.Errorf("Expected %s files in the tar.gz but received %s", strconv.Itoa(numFiles), strconv.Itoa(len(files)))
				}
			} else if err == nil {
				t.Errorf("Expected test %s to generate an error, but received nil.", tt.name)
			}
		})
	}
}

func TestSplitFiles(t *testing.T) {
	cr = &metricscfgv1beta1.MetricsConfig{}
	tmpDir := getTempDir(t, 0777, "./test_files", "tmp-*")
	defer os.RemoveAll(tmpDir)
	splitFilesTests := []struct {
		name          string
		dirMode       os.FileMode
		fileMode      os.FileMode
		files         []string
		expectedSplit bool
		maxBytes      int64
		expectErr     bool
	}{
		{
			name:          "test requires split",
			dirMode:       0777,
			fileMode:      0777,
			files:         []string{"ocp_node_label.csv"},
			maxBytes:      1 * 1024 * 1024,
			expectedSplit: true,
			expectErr:     false,
		},
		{
			name:          "test does not require split",
			dirMode:       0777,
			fileMode:      0777,
			files:         []string{"ocp_node_label.csv"},
			maxBytes:      100 * 1024 * 1024,
			expectedSplit: false,
			expectErr:     false,
		},
		{
			name:          "test failure to open file",
			dirMode:       0777,
			fileMode:      0000,
			files:         []string{"ocp_node_label.csv"},
			maxBytes:      1 * 1024 * 1024,
			expectedSplit: false,
			expectErr:     true,
		},
		{
			name:          "test bad csv read",
			dirMode:       0777,
			fileMode:      0777,
			files:         []string{"bad-csv.csv"},
			maxBytes:      512,
			expectedSplit: false,
			expectErr:     true,
		},
		{
			name:          "dir without write permissions",
			dirMode:       0555,
			fileMode:      0777,
			files:         []string{"ocp_node_label.csv"},
			maxBytes:      1 * 1024 * 1024,
			expectedSplit: false,
			expectErr:     true,
		},
		{
			name:          "small csv skips split",
			dirMode:       0777,
			fileMode:      0777,
			files:         []string{"ocp_node_label.csv", "small-csv.csv"},
			maxBytes:      1 * 1024 * 1024,
			expectedSplit: true,
			expectErr:     false,
		},
	}
	for _, tt := range splitFilesTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			dstTemp := getTempDir(t, 0777, tmpDir, "tmp-dir-*") // use 0777 so we can write file
			fileList := []os.FileInfo{}
			for _, file := range tt.files {
				fileInf, err := Copy(tt.fileMode, filepath.Join("test_files/", file), filepath.Join(dstTemp, file))
				if err != nil {
					t.Fatalf("%s failed to create file", tt.name)
				}
				fileList = append(fileList, fileInf)
			}
			setPerm(t, tt.dirMode, dstTemp) // now set the expected test permissions

			testPackager.maxBytes = tt.maxBytes
			files, split, err := testPackager.splitFiles(dstTemp, fileList)
			if tt.expectErr && err == nil {
				t.Errorf("%s expected an error but received nil", tt.name)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("%s did not expect error but got: %v", tt.name, err)
			}
			// make sure that the expected split matches
			if split != tt.expectedSplit {
				t.Errorf("Outcome for test %s:\nneedSplit Received: %v\nneedSPlit Expected: %v", tt.name, split, tt.expectedSplit)
			}
			// check the number of files created is more than the original if the split was required
			if len(files) > 0 && ((len(files) <= len(tt.files) && tt.expectedSplit) || (!tt.expectedSplit && len(files) != len(tt.files))) {
				t.Errorf("Outcome for test %s:\nOriginal number of files: %d\nResulting number of files: %d", tt.name, len(tt.files), len(files))
			}

			setPerm(t, 0777, dstTemp) // reset dir permissions for proper cleanup
		})
	}
}

func TestGetFileInfo(t *testing.T) {
	cr = &metricscfgv1beta1.MetricsConfig{}
	files := []string{
		"ff1c03d2-e303-4ab8-a8fc-d1267bf160d4_openshift_usage_report.0.csv",
		"ff1c03d2-e303-4ab8-a8fc-d1267bf160d4_openshift_usage_report.1.csv",
		"ff1c03d2-e303-4ab8-a8fc-d1267bf160d4_openshift_usage_report.2.csv",
		"ff1c03d2-e303-4ab8-a8fc-d1267bf160d4_openshift_usage_report.3.csv",
	}
	fileInfoTests := []struct {
		name             string
		tarFileName      string
		expectedManifest FileInfoManifest
		expectErr        bool
	}{
		{
			name:        "test good manifest",
			tarFileName: "20210720T180121-cost-mgmt.tar.gz",
			expectedManifest: FileInfoManifest{
				ClusterID: "30d0669b-4d46-479d-accb-9784e9d129d8",
				UUID:      "ff1c03d2-e303-4ab8-a8fc-d1267bf160d4",
				Files:     files,
			},
			expectErr: false,
		},
		{
			name:             "test bad tar file",
			tarFileName:      "badfile.tar.gz",
			expectedManifest: FileInfoManifest{},
			expectErr:        true,
		},
		{
			name:             "test nonexistent tar",
			tarFileName:      "nonexistent.tar.gz",
			expectedManifest: FileInfoManifest{},
			expectErr:        true,
		},
	}
	for _, tt := range fileInfoTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			manifest, err := testPackager.GetFileInfo(filepath.Join("test_files/", tt.tarFileName))
			if tt.expectErr && err == nil {
				t.Errorf("%s expected an error but received nil", tt.name)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("%s did not expect error but got: %v", tt.name, err)
			}
			if manifest.ClusterID != tt.expectedManifest.ClusterID {
				t.Errorf("%s expected %s as clusterID got %s", tt.name, tt.expectedManifest.ClusterID, manifest.ClusterID)
			}
			if manifest.UUID != tt.expectedManifest.UUID {
				t.Errorf("%s expected %s as manifestID got %s", tt.name, tt.expectedManifest.UUID, manifest.UUID)
			}
			if len(manifest.Files) != len(tt.expectedManifest.Files) {
				t.Errorf("%s expected %d files got %d files", tt.name, len(tt.expectedManifest.Files), len(manifest.Files))
			}
		})
	}
}

func TestTrimPackages(t *testing.T) {
	cr = &metricscfgv1beta1.MetricsConfig{}
	tmpDir := getTempDir(t, 0777, "./test_files", "tmp-*")
	defer os.RemoveAll(tmpDir)
	trimPackagesTests := []struct {
		name               string
		tmpFilePattern     string
		numFiles           int64
		maxReports         int64
		duplicateReports   bool
		numFilesExpected   int
		numReportsExpected int
		noPermDir          bool
		want               error
	}{
		{
			name:               "no reports in dir",
			tmpFilePattern:     "%d-*.csv",
			numFiles:           2,
			maxReports:         2,
			duplicateReports:   false,
			numFilesExpected:   2,
			numReportsExpected: 0,
			want:               nil,
		},
		{
			name:      "no permissions in dir",
			noPermDir: true,
			want:      errTest,
		},
		{
			name:               "no trimming needed",
			tmpFilePattern:     "%d-*.tar.gz",
			numFiles:           2,
			maxReports:         2,
			duplicateReports:   false,
			numFilesExpected:   2,
			numReportsExpected: 2,
			want:               nil,
		},
		{
			name:               "trimming needed",
			tmpFilePattern:     "%d-*.tar.gz",
			numFiles:           2,
			maxReports:         1,
			duplicateReports:   false,
			numFilesExpected:   1,
			numReportsExpected: 1,
			want:               nil,
		},
		{
			name:               "no trimming needed - split reports",
			tmpFilePattern:     "%d-*.tar.gz",
			numFiles:           2,
			maxReports:         2,
			duplicateReports:   true,
			numFilesExpected:   4,
			numReportsExpected: 2,
			want:               nil,
		},
		{
			name:               "trimming needed - split reports",
			tmpFilePattern:     "%d-*.tar.gz",
			numFiles:           2,
			maxReports:         1,
			duplicateReports:   true,
			numFilesExpected:   2,
			numReportsExpected: 1,
			want:               nil,
		},
	}

	for _, tt := range trimPackagesTests {
		t.Run(tt.name, func(t *testing.T) {
			var tmpDir2 string
			perms := os.FileMode(0777)

			if tt.noPermDir {
				tmpDir2 = getTempDir(t, 0200, tmpDir, "tmp-*")
			} else {
				tmpDir2 = getTempDir(t, perms, tmpDir, "tmp-*")
				for i := 0; i < int(tt.numFiles); i++ {
					_, err := os.CreateTemp(tmpDir2, fmt.Sprintf(tt.tmpFilePattern, i))
					if err != nil {
						t.Fatalf("failed to create temp file: %v", err)
					}
					if tt.duplicateReports {
						_, err := os.CreateTemp(tmpDir2, fmt.Sprintf(tt.tmpFilePattern, i))
						if err != nil {
							t.Fatalf("failed to create temp file: %v", err)
						}
					}
				}
			}
			defer os.RemoveAll(tmpDir2)

			dirCfg := &dirconfig.DirectoryConfig{
				Upload: dirconfig.Directory{Path: tmpDir2},
			}
			cr := &metricscfgv1beta1.MetricsConfig{}
			cr.Spec.Packaging.MaxReports = tt.maxReports
			testPackager := FilePackager{
				DirCfg: dirCfg,
			}
			got := testPackager.TrimPackages(cr)
			if tt.want == nil && got != nil {
				t.Errorf("%s did not expect error but got: %v", tt.name, got)
			}
			if tt.want != nil && got == nil {
				t.Errorf("%s expected an error but received nil", tt.name)
			}

			if tt.want == nil {
				files, err := dirCfg.Upload.GetFiles()
				if err != nil {
					t.Fatalf("%s: failed to read upload path: %v", tt.name, err)
				}
				if len(files) != tt.numFilesExpected {
					t.Errorf("%s expected %d files got %d files", tt.name, tt.numFilesExpected, len(files))
				}
				if cr.Status.Packaging.ReportCount != nil && *cr.Status.Packaging.ReportCount != int64(tt.numReportsExpected) {
					t.Errorf("%s expected %d number of reports got %d", tt.name, tt.numReportsExpected, *cr.Status.Packaging.ReportCount)
				}
			}
		})
	}
}

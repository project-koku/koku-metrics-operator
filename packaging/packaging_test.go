/*
Copyright 2020 Red Hat, Inc.
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.
You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package packaging

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
	"github.com/project-koku/korekuta-operator-go/dirconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var testingDir string
var dirCfg *dirconfig.DirectoryConfig = new(dirconfig.DirectoryConfig)
var log = zap.New()
var cost = &costmgmtv1alpha1.CostManagement{}
var testPackager = FilePackager{
	DirCfg: dirCfg,
	Log:    log,
	Cost:   cost,
}
var testManifest manifest
var testManifestInfo manifestInfo

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
	moving          testDirConfig
	split           testDirConfig
	tar             testDirConfig
	restricted      testDirConfig
	empty           testDirConfig
	restrictedEmpty testDirConfig
}

var testDirs testDirMap

func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func getTempFile(t *testing.T, mode os.FileMode, dir string) *os.File {
	tempFile, err := ioutil.TempFile(".", "garbage-file")
	if err != nil {
		t.Errorf("Failed to create temp file.")
	}
	if err := os.Chmod(tempFile.Name(), mode); err != nil {
		t.Errorf("Failed to change permissions of temp file.")
	}
	return tempFile
}
func getTempDir(t *testing.T, mode os.FileMode, dir, pattern string) string {
	tempDir, err := ioutil.TempDir(dir, pattern)
	if err != nil {
		t.Errorf("Failed to create temp folder.")
	}
	if err := os.Chmod(tempDir, mode); err != nil {
		t.Errorf("Failed to change permissions of temp file.")
	}
	return tempDir
}

func setup() error {
	type dirInfo struct {
		dirName  string
		files    []string
		dirMode  os.FileMode
		fileMode os.FileMode
	}
	testFiles := []string{"ocp_node_label.csv"}
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
			dirName:  "moving",
			files:    testFiles,
			dirMode:  0777,
			fileMode: 0777,
		},
		{
			dirName:  "split",
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
			files:    testFiles,
			dirMode:  0777,
			fileMode: 0000,
		},
		{
			dirName:  "empty",
			files:    nil,
			dirMode:  0777,
			fileMode: 0777,
		},
		{
			dirName: "restrictedEmpty",
			files:   nil,
			dirMode: 0000,
		},
	}
	// setup the initial testing directory
	fmt.Println("Setting up for packaging tests")
	testingUUID := uuid.New().String()
	testingDir = filepath.Join("../testing", testingUUID)
	if _, err := os.Stat(testingDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testingDir, os.ModePerm); err != nil {
			fmt.Println("Could not create testing directory")
			return err
		}
	}
	for _, directory := range dirInfoList {
		reportPath := filepath.Join(testingDir, directory.dirName)
		reportDataPath := filepath.Join(reportPath, "data")
		if _, err := os.Stat(reportPath); os.IsNotExist(err) {
			if err := os.Mkdir(reportPath, directory.dirMode); err != nil {
				fmt.Println("Could not create testing directory")
				return err
			}
			if directory.dirName != "empty" && directory.dirName != "restrictedEmpty" {
				if err := os.Mkdir(reportDataPath, directory.fileMode); err != nil {
					fmt.Println("Could not create testing directory")
					return err
				}
			}
			if !strings.Contains(directory.dirName, "restricted") && directory.dirName != "empty" {
				for _, reportFile := range directory.files {
					Copy(filepath.Join("../testfiles/", reportFile), filepath.Join(reportDataPath, reportFile))
				}
				os.Create(filepath.Join(reportDataPath, "nonCSV.txt"))
			}
			var fileList []os.FileInfo
			if !strings.Contains(directory.dirName, "restricted") && !strings.Contains(directory.dirName, "empty") {
				fileList, err = ioutil.ReadDir(reportDataPath)
				if err != nil {
					fmt.Println("Test files were not successfully created")
					return err
				}
			} else {
				fileList = nil
			}

			tmpDirMap := testDirConfig{
				directory: reportPath,
				files:     fileList,
			}
			if directory.dirName == "large" {
				testDirs.large = tmpDirMap
			} else if directory.dirName == "small" {
				testDirs.small = tmpDirMap
			} else if directory.dirName == "moving" {
				testDirs.moving = tmpDirMap
			} else if directory.dirName == "empty" {
				testDirs.empty = tmpDirMap
			} else if directory.dirName == "restricted" {
				testDirs.restricted = tmpDirMap
			} else if directory.dirName == "restrictedEmpty" {
				testDirs.restrictedEmpty = tmpDirMap
			} else if directory.dirName == "split" {
				testDirs.split = tmpDirMap
			} else if directory.dirName == "tar" {
				testDirs.tar = tmpDirMap
			}
		}
	}
	return nil
}

func shutdown() {
	fmt.Println("Tearing down for packaging tests")
	os.RemoveAll(testingDir)
}

func TestMain(m *testing.M) {
	err := setup()
	if err != nil {
		fmt.Println("Can not run tests because setup failed to create testing directories and files")
	} else {
		code := m.Run()
		shutdown()
		os.Exit(code)
	}
}

func TestNeedSplit(t *testing.T) {
	// create the needSplitTests
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
			want := make(map[int]string)
			fileInfoList, _ := ioutil.ReadDir(tt.dirName)
			for idx, file := range fileInfoList {
				// generate the expected file list
				if strings.HasSuffix(file.Name(), ".csv") {
					want[idx] = filepath.Join(tt.dirName, file.Name())
				}
			}
			for index, fileName := range want {
				if fileName != got[index] {
					t.Errorf("Expected %s but got %s", fileName, got[index])
				}
			}

		})
	}
}

func TestMoveFiles(t *testing.T) {
	// create the moveFiles tests
	moveFilesTests := []struct {
		name     string
		dirName  string
		fileUUID string
		want     []os.FileInfo
	}{
		{
			name:     "test moving dir",
			dirName:  testDirs.moving.directory,
			fileUUID: uuid.New().String(),
			want:     testDirs.large.files,
		},
		{
			name:     "test empty dir",
			dirName:  testDirs.empty.directory,
			fileUUID: uuid.New().String(),
			want:     nil,
		},
		{
			name:     "test restricted dir",
			dirName:  testDirs.restricted.directory,
			fileUUID: uuid.New().String(),
			want:     nil,
		},
	}
	for _, tt := range moveFilesTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			dirconfig.ParentDir = tt.dirName
			dirCfg.GetDirectoryConfig()
			testPackager.DirCfg = dirCfg
			testPackager.uid = tt.fileUUID
			got, _ := testPackager.moveFiles()
			if tt.want != nil {
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
			} else if got != nil {
				t.Errorf("Expected moved files to be nil")
			}
		})
	}
}

func TestPackagingReports(t *testing.T) {
	// create the packagingReports tests
	packagingReportTests := []struct {
		name          string
		dirName       string
		maxSize       int64
		want          error
		multipleFiles bool
	}{
		{
			name:          "test large dir",
			dirName:       testDirs.large.directory,
			maxSize:       1,
			multipleFiles: true,
			want:          nil,
		},
		{
			name:          "test small dir",
			dirName:       testDirs.small.directory,
			maxSize:       100,
			multipleFiles: false,
			want:          nil,
		},
		{
			name:          "test empty dir",
			dirName:       testDirs.empty.directory,
			maxSize:       100,
			multipleFiles: false,
			want:          errors.New("No reports found"),
		},
		{
			name:          "restricted",
			dirName:       testDirs.restrictedEmpty.directory,
			maxSize:       100,
			multipleFiles: false,
			want:          errors.New("Restricted"),
		},
		{
			name:          "nonexistent",
			dirName:       filepath.Join(testingDir, uuid.New().String()),
			maxSize:       100,
			multipleFiles: false,
			want:          errors.New("nonexistent"),
		},
	}
	for _, tt := range packagingReportTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			dirconfig.ParentDir = tt.dirName
			if tt.name != "restricted" {
				dirCfg.GetDirectoryConfig()
				testPackager.DirCfg = dirCfg
			} else {
				dirCfg := dirconfig.DirectoryConfig{
					Parent:  dirconfig.Directory{Path: tt.dirName},
					Upload:  dirconfig.Directory{Path: filepath.Join(tt.dirName, "upload")},
					Staging: dirconfig.Directory{Path: filepath.Join(tt.dirName, "staging")},
					Reports: dirconfig.Directory{Path: filepath.Join(tt.dirName, "reports")},
					Archive: dirconfig.Directory{Path: filepath.Join(tt.dirName, "archive")},
				}
				testPackager.DirCfg = &dirCfg
			}
			testPackager.MaxSize = tt.maxSize
			err := testPackager.PackageReports()
			if err == nil {
				outFiles, _ := testPackager.ReadUploadDir()
				if tt.multipleFiles && len(outFiles) <= 1 || (!tt.multipleFiles && len(outFiles) > 1) {
					t.Errorf("Outcome for test %s:\nReceived: %s\nExpected multpile files: %v", tt.name, strconv.Itoa(len(outFiles)), tt.multipleFiles)
				}
			} else {
				if tt.want == nil {
					t.Errorf("Expected no error but recieved one")
				}
				outFiles, err := testPackager.ReadUploadDir()
				if len(outFiles) != 0 {
					t.Errorf("An error occurred, but upload files were still generated.")
				}
				if err == nil {
					t.Errorf("Expected an error but recieved nil.")
				}
			}

		})
	}
}

func TestGetAndRenderManifest(t *testing.T) {
	// set up the tests to check the manifest contents
	getAndRenderManifestTests := []struct {
		name     string
		dirName  string
		fileList []os.FileInfo
	}{
		{
			name:     "test regular dir",
			dirName:  filepath.Join(testDirs.large.directory, "staging"),
			fileList: testDirs.large.files,
		},
		{
			name:     "test empty dir",
			dirName:  filepath.Join(testDirs.empty.directory, "staging"),
			fileList: testDirs.empty.files,
		},
	}
	for _, tt := range getAndRenderManifestTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			dirconfig.ParentDir = tt.dirName
			dirCfg.GetDirectoryConfig()
			testPackager.DirCfg = dirCfg
			// dirName = filepath.Join(tt.dirName, "staging")
			csvFileNames := testPackager.buildLocalCSVFileList(tt.fileList, tt.dirName)
			testPackager.getManifest(csvFileNames, tt.dirName)
			testPackager.manifest.renderManifest()
			// check that the manifest was generated correctly
			if testPackager.manifest.filename != filepath.Join(tt.dirName, "manifest.json") {
				t.Errorf("Manifest was not generated correctly")
			}
			// check that the manifest content is correct
			manifestData, _ := ioutil.ReadFile(testPackager.manifest.filename)
			var foundManifest manifest
			err := json.Unmarshal(manifestData, &foundManifest)
			if err != nil {
				t.Errorf("Error unmarshaling manifest")
			}
			// Define the expected manifest
			var expectedFiles []string
			for idx, _ := range csvFileNames {
				uploadName := testPackager.uid + "_openshift_usage_report." + strconv.Itoa(idx) + ".csv"
				expectedFiles = append(expectedFiles, uploadName)
			}
			manifestDate := metav1.Now()
			expectedManifest := manifest{
				UUID:      testPackager.uid,
				ClusterID: testPackager.Cost.Status.ClusterID,
				Version:   testPackager.Cost.Status.OperatorCommit,
				Date:      manifestDate.UTC().Format("2006-01-02 15:04:05"),
				Files:     expectedFiles,
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
			for _, file := range expectedFiles {
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

		})
	}
}

func TestRenderManifest(t *testing.T) {
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
			tarFileName:  filepath.Join(filepath.Join(testDirs.tar.directory, "upload"), "cost.tar.gz"),
			genCSVs:      true,
			expectedErr:  false,
		},
		{
			name:         "test mismatched files",
			dirName:      testDirs.large.directory,
			fileList:     testDirs.large.files,
			manifestName: "",
			tarFileName:  filepath.Join(filepath.Join(testDirs.large.directory, "upload"), "cost.tar.gz"),
			genCSVs:      true,
			expectedErr:  true,
		},
		{
			name:         "test empty dir",
			dirName:      testDirs.empty.directory,
			fileList:     testDirs.empty.files,
			manifestName: "",
			tarFileName:  filepath.Join(filepath.Join(testDirs.empty.directory, "upload"), "cost.tar.gz"),
			genCSVs:      true,
			expectedErr:  false,
		},
		{
			name:         "test nonexistant manifest path",
			dirName:      testDirs.large.directory,
			fileList:     testDirs.large.files,
			manifestName: testPackager.manifest.filename + "nonexistent",
			tarFileName:  filepath.Join(filepath.Join(testDirs.large.directory, "upload"), "cost.tar.gz"),
			genCSVs:      false,
			expectedErr:  true,
		},
		{
			name:         "test bad tarfile path",
			dirName:      testDirs.large.directory,
			fileList:     testDirs.large.files,
			manifestName: "",
			tarFileName:  filepath.Join(filepath.Join(filepath.Join(uuid.New().String(), testDirs.large.directory), "upload"), "cost-mgmt.tar.gz"),
			genCSVs:      true,
			expectedErr:  true,
		},
	}
	for _, tt := range writeTarballTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			dirconfig.ParentDir = tt.dirName
			dirCfg.GetDirectoryConfig()
			testPackager.DirCfg = dirCfg
			stagingDir := testPackager.DirCfg.Reports.Path
			var csvFileNames map[int]string
			if tt.genCSVs {
				csvFileNames = testPackager.buildLocalCSVFileList(tt.fileList, stagingDir)
			} else {
				csvFileNames = make(map[int]string)
			}
			var manifestName string
			testPackager.getManifest(csvFileNames, stagingDir)
			testPackager.manifest.renderManifest()
			if tt.manifestName == "" {
				manifestName = testPackager.manifest.filename
			} else {
				manifestName = tt.manifestName
			}
			err := testPackager.writeTarball(tt.tarFileName, manifestName, csvFileNames)
			// ensure the tarfile was created if we expect it to be
			if !tt.expectedErr {
				if _, err := os.Stat(tt.tarFileName); os.IsNotExist(err) {
					t.Errorf("Tar file was not created")
				}
				// the only testcases that should generate tars are the normal use case
				// and the empty use case
				var numFiles int
				if strings.Contains(tt.name, "empty") {
					// if the test case is the empty dir, there should only be a manifest
					numFiles = 1
				} else {
					// if the regular test case, there should be a manifest and 1 csv file
					numFiles = 2
				}
				// check the contents of the tarball
				file, err := os.Open(tt.tarFileName)
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
						fmt.Println(err)
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
	// create the buildLocalCSVFileList tests
	splitFilesTests := []struct {
		name          string
		dirName       string
		fileList      []os.FileInfo
		expectedSplit bool
		maxBytes      int64
		originalFiles int
		expectErr     bool
	}{
		{
			name:          "test requires split",
			dirName:       testDirs.split.directory,
			fileList:      testDirs.split.files,
			maxBytes:      1 * 1024 * 1024,
			expectedSplit: true,
			originalFiles: len(testDirs.split.files),
			expectErr:     false,
		},
		{
			name:          "test does not require split",
			dirName:       testDirs.split.directory,
			fileList:      testDirs.split.files,
			maxBytes:      100 * 1024 * 1024,
			expectedSplit: false,
			originalFiles: len(testDirs.split.files),
			expectErr:     false,
		},
		{
			name:          "test mismatched files",
			dirName:       testDirs.large.directory,
			fileList:      testDirs.large.files,
			maxBytes:      1 * 1024 * 1024,
			expectedSplit: true,
			originalFiles: len(testDirs.split.files),
			expectErr:     true,
		},
	}
	for _, tt := range splitFilesTests {
		// using tt.name from the case to use it as the `t.Run` test name
		t.Run(tt.name, func(t *testing.T) {
			dirconfig.ParentDir = tt.dirName
			dirCfg.GetDirectoryConfig()
			testPackager.DirCfg = dirCfg
			testPackager.maxBytes = tt.maxBytes
			files, split, err := testPackager.splitFiles(testPackager.DirCfg.Reports.Path, tt.fileList)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected an error but received nil")
				}
			} else {
				// make sure that the expected split matches
				if split != tt.expectedSplit {
					t.Errorf("Outcome for test %s:\nneedSplit Received: %v\nneedSPlit Expected: %v", tt.name, split, tt.expectedSplit)
				}
				// check the number of files created is more than the original if the split was required
				if (len(files) <= tt.originalFiles && tt.expectedSplit) || (!tt.expectedSplit && len(files) != tt.originalFiles) {
					t.Errorf("Outcome for test %s:\nOriginal number of files: %s\nResulting number of files: %s", tt.name, strconv.Itoa(tt.originalFiles), strconv.Itoa(len(files)))
				}
			}
		})
	}
}

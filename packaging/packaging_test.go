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
	"fmt"
	"github.com/project-koku/korekuta-operator-go/dirconfig"
	// "github.com/go-logr/logr"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"strconv"
	"strings"
	"testing"

	// uuidv4 "github.com/delaemon/go-uuidv4"
	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var DirMap = make(map[string][]os.FileInfo)
var testingDir string
var maxBytes int64
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

type mockPackager struct{}
func (p mockPackager) addFileToTarWriter(uploadName, filePath string, tarWriter *tar.Writer) error {
	return errors.New("This is a tar writer error")
}
func (p mockPackager) writeTarball(tarFileName, manifestFileName string, archiveFiles map[int]string) error {
	return testPackager.writeTarball(tarFileName, manifestFileName, archiveFiles)
}

// set up mocking scaffolding
// var marshalMock func(Manifest, string, string) ([]byte, error)

// type marshalCheckMock struct{}

// func (u marshalCheckMock) marshalFile(manifestStruct Manifest, space string, delimiter string) ([]byte, error) {
// 	return marshalMock(manifestStruct, space, delimiter)
// }

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
		t.Fatalf("Failed to create temp file.")
	}
	if err := os.Chmod(tempFile.Name(), mode); err != nil {
		t.Fatalf("Failed to change permissions of temp file.")
	}
	return tempFile
}
func getTempDir(t *testing.T, mode os.FileMode, dir, pattern string) string {
	tempDir, err := ioutil.TempDir(dir, pattern)
	if err != nil {
		t.Fatalf("Failed to create temp folder.")
	}
	if err := os.Chmod(tempDir, mode); err != nil {
		t.Fatalf("Failed to change permissions of temp file.")
	}
	return tempDir
}

func setup() {
	dirs := [3]string{"large", "small", "moving"}
	// setup the initial testing directory
	fmt.Println("Setting up for packaging tests")
	testingUUID := uuid.New().String()
	testingDir = "../testing/" + testingUUID
	if _, err := os.Stat(testingDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testingDir, os.ModePerm); err != nil {
			fmt.Println("Could not create testing directory")
		}
	}
	// setup a large/small dir within the testing directory
	// setup a data dir within each of the above directories and copy over the csv file to each respectively
	for _, reportSize := range dirs {
		reportPath := filepath.Join(testingDir, reportSize)
		reportDataPath := filepath.Join(reportPath, "data")
		if _, err := os.Stat(reportPath); os.IsNotExist(err) {
			if err := os.MkdirAll(reportPath, os.ModePerm); err != nil {
				fmt.Println("Could not create testing directory")
			}
			if err := os.MkdirAll(reportDataPath, os.ModePerm); err != nil {
				fmt.Println("Could not create testing directory")
			}
			Copy("../testfiles/ocp_pod_usage.csv", filepath.Join(reportDataPath, "ocp_pod_usage.csv"))
			Copy("../testfiles/ocp_node_label.csv", filepath.Join(reportDataPath, "ocp_node_label.csv"))
			os.Create(filepath.Join(reportDataPath, "nonCSV.txt"))
			fileList, err := ioutil.ReadDir(reportDataPath)
			if err != nil {
				fmt.Println("Something went wrong creating the test files")
			}
			DirMap[reportPath] = fileList
		}
	}
	// Initialize the filePackager
}

func shutdown() {
	fmt.Println("Tearing down for packaging tests")
	os.RemoveAll(testingDir)
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestNeedSplit(t *testing.T) {
	for dirName, fileList := range DirMap {
		if strings.Contains(dirName, "large") {
			// mimic a file that needs to be split by lowering the maxBytes
			maxBytes = 10 * 1024 * 1024
		} else {
			maxBytes = 100 * 1024 * 1024
		}
		needSplit := testPackager.needSplit(fileList, maxBytes)
		var expectedNeedSplit bool
		// if we are looking at the large csv we expect to need to split
		if strings.Contains(dirName, "large") {
			expectedNeedSplit = true
		} else {
			// but if we are looking at the small one we should not require a split
			expectedNeedSplit = false
		}
		if needSplit != expectedNeedSplit {
			t.Fatalf("Expected %v but got %v", expectedNeedSplit, needSplit)
		}
	}
}

func TestBuildLocalCSVFileList(t *testing.T) {
	for dirName, fileList := range DirMap {
		dirName = filepath.Join(dirName, "data")
		expectedMap := make(map[int]string)
		fileList := testPackager.buildLocalCSVFileList(fileList, dirName)
		fileInfoList, _ := ioutil.ReadDir(dirName)
		for idx, file := range fileInfoList {
			// generate the expected file list
			if strings.HasSuffix(file.Name(), ".csv"){
				expectedMap[idx] = filepath.Join(dirName, file.Name())
			}
		}
		// compare the file list received to the one we expect
		for index, fileName := range expectedMap {
			if fileName != fileList[index] {
				t.Fatalf("Expected %s but got %s", fileName, fileList[index])
			}
		}
	}
}

func TestMoveFiles(t *testing.T) {
	// log := zap.New()
	fileUUID := uuid.New().String()
	// cost := &costmgmtv1alpha1.CostManagement{}
	for dirName, _ := range DirMap {
		// Only test this on the moving test dir so we don't destroy all data
		if strings.Contains(dirName, "moving") {
			dirconfig.ParentDir = dirName
			dirCfg.GetDirectoryConfig()
			testPackager.DirCfg = dirCfg
			testPackager.uid = fileUUID
			movedFiles, _ := testPackager.moveFiles()
			for _, file := range movedFiles {
				// check that the file contains the uuid
				if !strings.Contains(file.Name(), fileUUID) {
					t.Fatalf("Expected %s to be in the name but got %s", fileUUID, file.Name())
				}
				// check that the file exists in the staging directory
				if _, err := os.Stat(filepath.Join(filepath.Join(dirName, "staging"), file.Name())); os.IsNotExist(err) {
					t.Fatalf("File does not exist in the staging directory")
				}
			}
			// test MoveFiles when the FileList is empty (create an empty dir)
			if err := os.MkdirAll(filepath.Join(dirName, "emptyDir"), os.ModePerm); err != nil {
				fmt.Println("Could not create empty directory")
			}
			dirconfig.ParentDir = filepath.Join(dirName, "emptyDir")
			dirCfg.GetDirectoryConfig()
			testPackager.DirCfg = dirCfg
			movedFiles, _ = testPackager.moveFiles()
			if movedFiles != nil {
				t.Fatalf("Found files in an empty directory")
			}
		}
	}
}

func TestPackagingReports(t *testing.T) {
	var maxSize int64
	for dirName, _ := range DirMap {
		if !strings.Contains(dirName, "moving") {
			if strings.Contains(dirName, "large") {
				// mimic a file that needs to be split by lowering the maxSize
				maxSize = 10
			} else {
				maxSize = 100
			}
			dirconfig.ParentDir = dirName
			dirCfg.GetDirectoryConfig()
			testPackager.DirCfg = dirCfg
			testPackager.PackageReports(maxSize)
			// test the Read upload dir function here
			outFiles, _ := testPackager.ReadUploadDir()
			if strings.Contains(dirName, "small") {
				if len(outFiles) > 1 {
					t.Fatalf("Too many files generated")
				}
			} else {
				if len(outFiles) < 4 {
					t.Fatalf("Not enough files generated")
				}
			}
		}
	}
}

func TestGetAndRenderManifest(t *testing.T) {
	for dirName, fileList := range DirMap {
		dirconfig.ParentDir = dirName
		dirCfg.GetDirectoryConfig()
		testPackager.DirCfg = dirCfg
		dirName = filepath.Join(dirName, "staging")
		csvFileNames := testPackager.buildLocalCSVFileList(fileList, dirName)
		testPackager.getManifest(csvFileNames, dirCfg.Staging.Path)
		testPackager.manifest.renderManifest()
		// check that the manifest was generated correctly
		if testPackager.manifest.filename != filepath.Join(dirName, "manifest.json") {
			t.Fatalf("Manifest was not generated correctly")
		}
		// check that the manifest content is correct
		manifestData, _ := ioutil.ReadFile(testPackager.manifest.filename)
		var foundManifest manifest
		err := json.Unmarshal(manifestData, &foundManifest)
		if err != nil {
			t.Fatalf("Error unmarshaling manifest")
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
			t.Fatalf(errorMsg, expectedManifest.UUID, foundManifest.UUID)
		}
		if foundManifest.ClusterID != expectedManifest.ClusterID {
			t.Fatalf(errorMsg, expectedManifest.ClusterID, foundManifest.ClusterID)
		}
		if foundManifest.Version != expectedManifest.Version {
			t.Fatalf(errorMsg, expectedManifest.Version, foundManifest.Version)
		}
		for _, file := range expectedFiles {
			found := false
			for _, foundFile := range foundManifest.Files {
				if file == foundFile {
					found = true
				}
			}
			if !found {
				t.Fatalf(errorMsg, file, foundManifest.Files)
			}
		}
	
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
	for dirName, fileList := range DirMap {
		dirconfig.ParentDir = dirName
		dirCfg.GetDirectoryConfig()
		testPackager.DirCfg = dirCfg
		uploadDir := testPackager.DirCfg.Upload.Path
		dirName = filepath.Join(dirName, "staging")
		csvFileNames := testPackager.buildLocalCSVFileList(fileList, dirName)
		testPackager.getManifest(csvFileNames, dirCfg.Staging.Path)
		testPackager.manifest.renderManifest()
		tarFileName := filepath.Join(uploadDir, "cost-mgmt-test.tar.gz")
		testPackager.writeTarball(tarFileName, testPackager.manifest.filename, csvFileNames)
		// ensure the tarfile was created
		if _, err := os.Stat(tarFileName); os.IsNotExist(err) {
			t.Fatalf("Tar file was not created")
		}
		// TODO: check the contents of the tarfile
	}
}

func TestWriteTarballError(t *testing.T) {
	for dirName, fileList := range DirMap {
		dirconfig.ParentDir = dirName
		dirCfg.GetDirectoryConfig()
		testPackager.DirCfg = dirCfg
		uploadDir := testPackager.DirCfg.Upload.Path
		dirName = filepath.Join(dirName, "staging")
		csvFileNames := testPackager.buildLocalCSVFileList(fileList, dirName)
		testPackager.getManifest(csvFileNames, dirCfg.Staging.Path)
		testPackager.manifest.renderManifest()
		tarFileName := filepath.Join(uploadDir, "cost-mgmt-test.tar.gz")
		// testPackager.writeTarball(tarFileName, testPackager.manifest.filename, csvFileNames)
		mockPackager.writeTarball(testPackager, tarFileName, testPackager.manifest.filename, csvFileNames)
		// ensure the tarfile was created
		if _, err := os.Stat(tarFileName); os.IsNotExist(err) {
			t.Fatalf("Tar file was not created")
		}
		// TODO: check the contents of the tarfile
	}
}

// func TestSplitFiles(t *testing.T) {
// 	for dirName, fileList := range DirMap {
// 		dirconfig.ParentDir = dirName
// 		dirCfg.GetDirectoryConfig()
// 		testPackager.DirCfg = dirCfg
// 		var expectedSplit bool
// 		if strings.Contains(dirName, "large") {
// 			// mimic a file that needs to be split by lowering the maxBytes
// 			maxBytes = 10 * 1024 * 1024
// 			expectedSplit = true
// 		} else {
// 			maxBytes = 100 * 1024 * 1024
// 			expectedSplit = false
// 		}
// 		// dirName = filepath.Join(dirName, "staging")
// 		files, split, err := testPackager.splitFiles(testPackager.DirCfg.Staging.Path, fileList, maxBytes)
// 		fmt.Println(expectedSplit)
// 		fmt.Println(split)
// 		fmt.Println(err)
// 		// if expectedSplit != split {
// 		// 	t.Fatalf("Expected %v but recieved %v", expectedSplit, split)
// 		// }
// 		fmt.Println("################################*************************** FILES: ")
// 		fmt.Println(files)
// 		for _, file := range files {
// 			fmt.Println(file.Name())
// 		}
// 	}
// }

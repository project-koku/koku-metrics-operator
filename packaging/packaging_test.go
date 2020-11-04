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
	"fmt"
	"github.com/project-koku/korekuta-operator-go/dirconfig"
	// "github.com/go-logr/logr"
	"github.com/google/uuid"
	"io"
	"io/ioutil"
	"encoding/json"
	"errors"
	"os"
	"path"
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

// set up mocking scaffolding 
var marshalMock func(Manifest, string, string) ([]byte, error)
type marshalCheckMock struct{}
func (u marshalCheckMock) marshalFile(manifestStruct Manifest, space string, delimiter string) ([]byte, error) {
	return marshalMock(manifestStruct, space, delimiter)
}

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
		reportPath := path.Join(testingDir, reportSize)
		reportDataPath := path.Join(reportPath, "data")
		if _, err := os.Stat(reportPath); os.IsNotExist(err) {
			if err := os.MkdirAll(reportPath, os.ModePerm); err != nil {
				fmt.Println("Could not create testing directory")
			}
			if err := os.MkdirAll(reportDataPath, os.ModePerm); err != nil {
				fmt.Println("Could not create testing directory")
			}
			Copy("../testfiles/ocp_pod_usage.csv", path.Join(reportDataPath, "ocp_pod_usage.csv"))
			fileList, err := ioutil.ReadDir(reportDataPath)
			if err != nil {
				fmt.Println("Something went wrong creating the test files")
			}
			DirMap[reportPath] = fileList
		}
	}
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
		needSplit := NeedSplit(fileList, maxBytes)
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
		dirName = path.Join(dirName, "data")
		var expectedList []string
		fileList := BuildLocalCSVFileList(fileList, dirName)
		fileInfoList, _ := ioutil.ReadDir(dirName)
		for _, file := range fileInfoList {
			// generate the expected file list
			expectedList = append(expectedList, path.Join(dirName, file.Name()))
		}
		// compare the file list received to the one we expect
		for index, fileName := range expectedList {
			if fileName != fileList[index] {
				t.Fatalf("Expected %s but got %s", fileName, fileList[index])
			}
		}
	}
}

func TestMoveFiles(t *testing.T) {
	log := zap.New()
	fileUUID := uuid.New().String()
	cost := &costmgmtv1alpha1.CostManagement{}
	for dirName, _ := range DirMap {
		// Only test this on the moving test dir so we don't destroy all data 
		if strings.Contains(dirName, "moving") {
			dirconfig.ParentDir = dirName
			dirCfg.GetDirectoryConfig()
			movedFiles, _ := MoveFiles(log, dirCfg.Reports, dirCfg.Staging, cost, fileUUID)
			for _, file := range movedFiles {
				// check that the file contains the uuid 
				if !strings.Contains(file.Name(), fileUUID){
					t.Fatalf("Expected %s to be in the name but got %s", fileUUID, file.Name())
				}
				// check that the file exists in the staging directory
				if _, err := os.Stat(path.Join(path.Join(dirName, "staging"), file.Name())); os.IsNotExist(err) {
					t.Fatalf("File does not exist in the staging directory")
				}
			}
			// test MoveFiles when the FileList is empty (create an empty dir)
			if err := os.MkdirAll(path.Join(dirName, "emptyDir"), os.ModePerm); err != nil {
				fmt.Println("Could not create empty directory")
			}
			dirconfig.ParentDir = path.Join(dirName, "emptyDir")
			dirCfg.GetDirectoryConfig()
			movedFiles, _ = MoveFiles(log, dirCfg.Reports, dirCfg.Staging, cost, fileUUID)
			if movedFiles != nil {
				t.Fatalf("Found files in an empty directory")
			}
		}
	}
}

func TestPackagingReports(t *testing.T) {
	log := zap.New()
	cost := &costmgmtv1alpha1.CostManagement{}
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
			uploadDir, _ := PackageReports(log, dirCfg, cost, maxSize)
			fmt.Println(uploadDir)
			// test the Read upload dir function here 
			outFiles, _ := ReadUploadDir(dirCfg)
			if strings.Contains(dirName, "small"){
				if len(outFiles) > 1{
					t.Fatalf("Too many files generated")
				} 
			} else {
				if len(outFiles) < 4{
					t.Fatalf("Not enough files generated")
				}
			}
		}
	}
}

func TestRenderManifest(t *testing.T) {
	// TODO: fix me this is a stub
	log := zap.New()
	fileUUID := uuid.New().String()
	cost := &costmgmtv1alpha1.CostManagement{}
	for dirName, fileList := range DirMap {
		if strings.Contains(dirName, "small") {
			// test the marshal error
			marshal = marshalCheckMock{}
			marshalMock = func(manifestFile Manifest, space string, delimiter string) ([]byte, error){
				marshalError := errors.New("Error marshaling occurred")
				var byteArray []byte
				return byteArray, marshalError
			}
			dirName = path.Join(dirName, "data")
			csvFileNames := BuildLocalCSVFileList(fileList, dirName)
			_, err := RenderManifest(log, csvFileNames, cost, dirName, fileUUID)
			if err == nil {
				t.Fatalf("Marshal mock did not raise an error as expected!")
			}
		} else {
			dirName = path.Join(dirName, "data")
			csvFileNames := BuildLocalCSVFileList(fileList, dirName)
			manifestName, _ := RenderManifest(log, csvFileNames, cost, dirName, fileUUID)
			// check that the manifest was generated correctly
			if manifestName != path.Join(dirName, "manifest.json") {
				t.Fatalf("Manifest was not generated correctly")
			}
			// check that the manifest content is correct
			manifestData, _ := ioutil.ReadFile(manifestName)
			var foundManifest Manifest 
			err := json.Unmarshal(manifestData, &foundManifest)
			if err != nil {
				t.Fatalf("Error unmarshaling manifest")
			}
			// Define the expected manifest 
			var expectedFiles []string
			for idx := range csvFileNames {
				uploadName := fileUUID + "_openshift_usage_report." + strconv.Itoa(idx) + ".csv"
				expectedFiles = append(expectedFiles, uploadName)
			}
			manifestDate := metav1.Now()
			expectedManifest := Manifest{
				UUID:      fileUUID,
				ClusterID: cost.Status.ClusterID,
				Version:   cost.Status.OperatorCommit,
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
			for index, file := range expectedFiles {
				if file != foundManifest.Files[index] {
					t.Fatalf(errorMsg, file, foundManifest.Files[index])
				}
			}
		}
	}
}

func TestWriteTarball(t *testing.T) {
	log := zap.New()
	fileUUID := uuid.New().String()
	cost := &costmgmtv1alpha1.CostManagement{}
	for dirName, fileList := range DirMap {
		uploadDir := path.Join(dirName, "upload")
		dirName = path.Join(dirName, "staging")
		csvFileNames := BuildLocalCSVFileList(fileList, dirName)
		manifestName, _ := RenderManifest(log, csvFileNames, cost, dirName, fileUUID)
		tarFileName := path.Join(uploadDir, "cost-mgmt-test.tar.gz")
		WriteTarball(log, tarFileName, manifestName, fileUUID, csvFileNames)
		// ensure the tarfile was created
		if _, err := os.Stat(tarFileName); os.IsNotExist(err) {
			t.Fatalf("Tar file was not created")
		  }
		// TODO: check the contents of the tarfile
	}
}

func TestSplitFiles(t *testing.T) {
	log := zap.New()
	for dirName, fileList := range DirMap {
		if strings.Contains(dirName, "large") {
			// mimic a file that needs to be split by lowering the maxBytes
			maxBytes = 10 * 1024 * 1024
		} else {
			maxBytes = 100 * 1024 * 1024
		}
		dirName = path.Join(dirName, "data")
		files, _ := SplitFiles(log, dirName, fileList, maxBytes)
		fmt.Println("################################*************************** FILES: ")
		fmt.Println(files)
		for _, file := range files{
			fmt.Println(file.Name())
		}
	}
}
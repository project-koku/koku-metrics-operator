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
	"os"
	"path"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"strings"
	"testing"

	// uuidv4 "github.com/delaemon/go-uuidv4"
	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
)

var DirMap = make(map[string][]os.FileInfo)
var testingDir string
var maxBytes int64 = 100 * 1024 * 1024
var dirCfg *dirconfig.DirectoryConfig = new(dirconfig.DirectoryConfig)

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
	// setup a data dir within each of the above directories and copy over a large and small CSV file to each respectively
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
			// if reportSize == "large" {
			// 	Copy("../testfiles/ocp_pod_usage_large.csv", path.Join(reportDataPath, "ocp_pod_usage-1.csv"))
			// } else {
			// 	Copy("../testfiles/ocp_pod_usage.csv", path.Join(reportDataPath, "ocp_pod_usage-1.csv"))
			// }
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

func TestReadUploadDir(t *testing.T) {
	// fixme
	outFiles, _ := ReadUploadDir(dirCfg)
	fmt.Println(outFiles)
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
				// check that the data directory has been removed
				if _, err := os.Stat(path.Join(dirName, "data")); !os.IsNotExist(err) {
					t.Fatalf("Expected %s to not exist", path.Join(dirName, "data"))
				}
				// check that the file contains the uuid 
				if !strings.Contains(file.Name(), fileUUID){
					t.Fatalf("Expected %s to be in the name but got %s", fileUUID, file.Name())
				}
				// check that the file exists in the staging directory
				if _, err := os.Stat(path.Join(path.Join(dirName, "staging"), file.Name())); os.IsNotExist(err) {
					t.Fatalf("File does not exist in the staging directory")
				}
			}
		}
	}
}

func TestPackagingReports(t *testing.T) {
	log := zap.New()
	cost := &costmgmtv1alpha1.CostManagement{}
	var maxSize int64 = 100
	for dirName, _ := range DirMap {
		if !strings.Contains(dirName, "moving") {
			if strings.Contains(dirName, "large") {
				// mimic a file that needs to be split by lowering the maxSize
				maxSize = 10
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

func TestSplitFiles(t *testing.T) {
	// TODO: fix me this is a stub
	log := zap.New()
	for dirName, fileList := range DirMap {
		if strings.Contains(dirName, "large") {
			// mimic a file that needs to be split by lowering the maxBytes
			maxBytes = 10 * 1024 * 1024
		}
		dirName = path.Join(dirName, "data")
		files, _ := SplitFiles(log, dirName, fileList, maxBytes)
		fmt.Println(files)
	}
}

func TestRenderManifest(t *testing.T) {
	// TODO: fix me this is a stub
	log := zap.New()
	fileUUID := uuid.New().String()
	cost := &costmgmtv1alpha1.CostManagement{}
	for dirName, fileList := range DirMap {
		dirName = path.Join(dirName, "data")
		csvFileNames := BuildLocalCSVFileList(fileList, dirName)
		manifestName, _ := RenderManifest(log, csvFileNames, cost, dirName, fileUUID)
		// check that the manifest was generated correctly
		if manifestName != path.Join(dirName, "manifest.json") {
			t.Fatalf("Manifest was not generated correctly")
		}
		// check that the manifest content is correct
		
	}
}

func TestWriteTarball(t *testing.T) {
	//TODO: fix me this is a stub
	log := zap.New()
	fileUUID := uuid.New().String()
	cost := &costmgmtv1alpha1.CostManagement{}
	for dirName, fileList := range DirMap {
		dirName = path.Join(dirName, "data")
		csvFileNames := BuildLocalCSVFileList(fileList, dirName)
		manifestName, _ := RenderManifest(log, csvFileNames, cost, dirName, fileUUID)
		WriteTarball(log, "cost-mgmt.tar.gz", manifestName, fileUUID, csvFileNames)
	}
}

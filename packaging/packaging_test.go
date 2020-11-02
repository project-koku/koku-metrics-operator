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

// func TestSplit(t *testing.T) {
// 	var log logr.Logger
// 	cost := &costmgmtv1alpha1.CostManagement{}
// 	uploadDir, err := Split(log, "/tmp/cost-mgmt-operator-reports", cost, 100)
// 	if err != nil {
// 		log.Info("something went wrong!")
// 	}
// 	var expectedUploadDir = "/tmp/cost-mgmt-operator-reports/upload"

// 	if uploadDir != expectedUploadDir {
// 		t.Fatalf("Expected %s but got %s", expectedUploadDir, uploadDir)
// 	}
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

func setup() {
	// var log logr.Logger
	dirs := [2]string{"large", "small"}
	testingUUID := uuid.New().String()
	testingDir := "../testing/" + testingUUID
	if _, err := os.Stat(testingDir); os.IsNotExist(err) {
		if err := os.MkdirAll(testingDir, os.ModePerm); err != nil {
			fmt.Println("Could not create testing directory")
		}
	}
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
			if reportSize == "large" {
				Copy("/Users/ashleybrookeaiken/Development/nise/October-2020-505a2749-0f02-4ad1-a46a-6216ca3684fc-ocp_pod_usage-1.csv", path.Join(reportDataPath, "ocp_pod_usage-1.csv"))
			} else {
				Copy("/Users/ashleybrookeaiken/Development/nise/October-2020-123abc3452323-ocp_pod_usage-1.csv", path.Join(reportDataPath, "ocp_pod_usage-1.csv"))
			}
			fileList, err := ioutil.ReadDir(reportDataPath)
			if err != nil {
				fmt.Println("Something went wrong creating the test files")
			}
			DirMap[reportDataPath] = fileList
		}

		fmt.Println("Setting up for packaging tests")
	}
}

func shutdown() {
	// var log logr.Logger
	// uncomment this later
	// for _, dirName := range DirArray {
	// os.RemoveAll(testingDir)
	// }
	fmt.Println("Tearing down for packaging tests")
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func TestNeedSplit(t *testing.T) {
	var maxBytes int64 = 100 * 1024 * 1024
	for dirName, fileList := range DirMap {
		needSplit := NeedSplit(fileList, maxBytes)
		var expectedNeedSplit bool
		if strings.Contains(dirName, "large") {
			expectedNeedSplit = true
		} else {
			expectedNeedSplit = false
		}
		if needSplit != expectedNeedSplit {
			t.Fatalf("Expected %v but got %v", expectedNeedSplit, needSplit)
		}
	}
}

func TestBuildLocalCSVFileList(t *testing.T) {
	for dirName, fileList := range DirMap {
		var expectedList []string
		fileList := BuildLocalCSVFileList(fileList, dirName)
		fileInfoList, _ := ioutil.ReadDir(dirName)
		for _, file := range fileInfoList {
			expectedList = append(expectedList, path.Join(dirName, file.Name()))
		}
		for index, fileName := range expectedList {
			if fileName != fileList[index] {
				t.Fatalf("Expected %s but got %s", fileName, fileList[index])
			}
		}
	}
}

func TestMoveFiles(t *testing.T) {
	// var log *logr.Logger = new(logr.Logger)
	// log := Log.WithValues("costmanagement", "Tests")
	log := zap.New()
	fileUUID := uuid.New().String()
	var dirCfg *dirconfig.DirectoryConfig = new(dirconfig.DirectoryConfig)
	cost := &costmgmtv1alpha1.CostManagement{}
	movedFiles, _ := MoveFiles(log, dirCfg.Reports, dirCfg.Staging, cost, fileUUID)
	fmt.Println(movedFiles)
	// TODO: fix this later - add an assert and make the dirconfig point to test dirs
	// for dirName, _ := range DirMap {
	// 	MoveFiles(log, dirCfg.Reports, dirCfg.Staging, cost, fileUUID)
	// }
}

func TestPackagingReports(t *testing.T) {
	// test the packagingreports function
	// TODO: fixme this is a stub
	log := zap.New()
	var dirCfg *dirconfig.DirectoryConfig = new(dirconfig.DirectoryConfig)
	cost := &costmgmtv1alpha1.CostManagement{}
	for dirName, _ := range DirMap {
		parentDir := dirconfig.Directory{Path: dirName}
		dirCfg.Parent = parentDir
		uploadDir, _ := PackageReports(log, dirCfg, cost, 100)
		fmt.Println(uploadDir)
	}
}

func TestReadUploadDir(t *testing.T) {
	// test reading the upload directory
	// TODO: fixme this is a stub
	var dirCfg *dirconfig.DirectoryConfig = new(dirconfig.DirectoryConfig)
	outFiles, _ := ReadUploadDir(dirCfg)
	fmt.Println(outFiles)
}

func TestSplitFiles(t *testing.T) {
	// TODO: fix me this is a stub
	log := zap.New()
	var maxBytes int64 = 100 * 1024 * 1024
	for dirName, fileList := range DirMap {
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
		csvFileNames := BuildLocalCSVFileList(fileList, dirName)
		manifestName, _ := RenderManifest(log, csvFileNames, cost, dirName, fileUUID)
		fmt.Println(manifestName)
	}
}

func TestWriteTarball(t *testing.T) {
	//TODO: fix me this is a stub
	log := zap.New()
	fileUUID := uuid.New().String()
	cost := &costmgmtv1alpha1.CostManagement{}
	for dirName, fileList := range DirMap {
		csvFileNames := BuildLocalCSVFileList(fileList, dirName)
		manifestName, _ := RenderManifest(log, csvFileNames, cost, dirName, fileUUID)
		WriteTarball(log, "cost-mgmt.tar.gz", manifestName, fileUUID, csvFileNames)
	}
}

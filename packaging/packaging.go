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
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	uuidv4 "github.com/delaemon/go-uuidv4"
	"github.com/go-logr/logr"
	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
	"github.com/project-koku/korekuta-operator-go/dirconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Define the global variables
var megaByte int64 = 1024 * 1024

// the csv module doesn't expose the bytes-offset of the
// underlying file object.
// instead, the script estimates the size of the data as VARIANCE percent larger than a
// naïve string concatenation of the CSV fields to cover the overhead of quoting
// and delimiters. This gets close enough for now.
// VARIANCE := 0.03
var variance float64 = 0.03

// if we're creating more than 1k files, something is probably wrong.
var maxSplits int64 = 1000

// Manifest template
type Manifest struct {
	UUID      string   `json:"uuid"`
	ClusterID string   `json:"cluster_id"`
	Version   string   `json:"version"`
	Date      string   `json:"date"`
	Files     []string `json:"files"`
}

// ErrNoReports a "no reports" Error type
var ErrNoReports = errors.New("reports not found")

// BuildLocalCSVFileList gets the list of files in the staging directory
func BuildLocalCSVFileList(fileList []os.FileInfo, stagingDirectory string) ([]string, error) {
	var csvList []string
	for _, file := range fileList {
		if strings.Contains(file.Name(), ".csv") {
			csvList = append(csvList, path.Join(stagingDirectory, file.Name()))
		}
	}
	return csvList, nil
}

// NeedSplit determines if any of the files to be packaged need to be split.
func NeedSplit(filepath string, maxBytes int64) ([]os.FileInfo, bool, error) {
	var totalSize int64 = 0
	fileList, err := ioutil.ReadDir(filepath)
	if err != nil {
		return nil, false, fmt.Errorf("NeedSplit: failed to read directory: %v", err)
	}
	for _, file := range fileList {
		fileSize := file.Size()
		totalSize += fileSize
		if fileSize >= maxBytes || totalSize >= maxBytes {
			return fileList, true, nil
		}
	}
	return fileList, false, nil
}

// RenderManifest writes the manifest
func RenderManifest(logger logr.Logger, archiveFiles []string, cost *costmgmtv1alpha1.CostManagement, filepath string) (string, string, error) {
	log := logger.WithValues("costmanagement", "RenderManifest")
	// setup the manifest
	manifestUUID, _ := uuidv4.Generate()
	manifestDate := metav1.Now()
	var manifestFiles []string
	for idx := range archiveFiles {
		uploadName := manifestUUID + "_openshift_usage_report." + strconv.Itoa(idx) + ".csv"
		manifestFiles = append(manifestFiles, uploadName)
	}
	fileManifest := Manifest{
		UUID:      manifestUUID,
		ClusterID: cost.Status.ClusterID,
		Version:   cost.Status.OperatorCommit,
		Date:      manifestDate.UTC().Format("2006-01-02 15:04:05"),
		Files:     manifestFiles,
	}
	manifestFileName := path.Join(filepath, "manifest.json")
	// write the manifest file
	file, err := json.MarshalIndent(fileManifest, "", " ")
	if err != nil {
		return "", "", fmt.Errorf("RenderManifest: failed to marshal manifest: %v", err)
	}
	if err := ioutil.WriteFile(manifestFileName, file, 0644); err != nil {
		return "", "", fmt.Errorf("RenderManifest: failed to write manifest: %v", err)
	}
	// return the manifest file/uuid
	log.Info("Generated manifest file", "manifest", manifestFileName)
	return manifestFileName, manifestUUID, nil
}

func addFileToTarWriter(logger logr.Logger, uploadName, filePath string, tarWriter *tar.Writer) error {
	log := logger.WithValues("costmanagement", "addFileToTarWriter")
	log.Info("Adding file to tar.gz", "file", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    uploadName,
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if _, err := io.Copy(tarWriter, file); err != nil {
		return err
	}

	return nil
}

// WriteTarball packages the files into tar balls
func WriteTarball(logger logr.Logger, tarFileName, manifestFileName, manifestUUID string, archiveFiles []string, fileNum ...int) error {
	index := 0
	if len(fileNum) > 0 {
		index = fileNum[0]
	}
	if len(archiveFiles) <= 0 {
		fmt.Println("no files to add to tar file")
		return ErrNoReports
	}
	// create the tarfile
	tarFile, err := os.Create(tarFileName)
	if err != nil {
		fmt.Println("error creating tar file")
		return err
	}
	defer tarFile.Close()

	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()

	tw := tar.NewWriter(gzipWriter)
	defer tw.Close()

	// add the files to the tarFile
	for idx, fileName := range archiveFiles {
		if index != 0 {
			idx = index
		}
		fmt.Println(fileName)
		if strings.Contains(fileName, ".csv") {
			uploadName := manifestUUID + "_openshift_usage_report." + strconv.Itoa(idx) + ".csv"
			fmt.Println(uploadName)
			err := addFileToTarWriter(logger, uploadName, fileName, tw)
			if err != nil {
				fmt.Println("error creating the tar file")
				return err
			}
		}
	}
	err = addFileToTarWriter(logger, "manifest.json", manifestFileName, tw)
	if err != nil {
		fmt.Println("error creating the tar file")
		return err
	}
	return nil

}

// WritePart writes a portion of a split file into a new file
func WritePart(logger logr.Logger, fileName string, csvReader *csv.Reader, csvHeader []string, num int64, maxBytes int64) (string, bool, error) {
	log := logger.WithValues("costmanagement", "WritePart")
	fileNamePart := strings.TrimSuffix(fileName, ".csv")
	sizeEstimate := 0
	splitFileName := fileNamePart + strconv.FormatInt(num, 10) + ".csv"
	log.Info("Creating file ", "file", splitFileName)
	splitFile, err := os.Create(splitFileName)
	if err != nil {
		return "", false, fmt.Errorf("WritePart: error creating file: %v", err)
	}
	// Create the csv writer
	writer := csv.NewWriter(splitFile)
	// Preserve the header
	writer.Write(csvHeader)
	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			writer.Flush()
			return splitFileName, true, nil
		} else if err != nil {
			return "", false, err
		}
		writer.Write(row)
		rowLen := len(strings.Join(row, ","))
		rowSize := rowLen + int(float64(rowLen)*variance)
		sizeEstimate += rowSize
		if sizeEstimate >= int(maxBytes) {
			writer.Flush()
			return splitFileName, false, nil
		}
	}
}

// SplitFiles breaks larger files into smaller ones
func SplitFiles(logger logr.Logger, filePath string, fileList []os.FileInfo, maxBytes int64) ([]os.FileInfo, error) {
	var splitFiles []os.FileInfo
	for _, file := range fileList {
		absPath := path.Join(filePath, file.Name())
		fileSize := file.Size()
		if fileSize >= maxBytes {
			// open the file
			csvFile, err := os.Open(absPath)
			if err != nil {
				return nil, fmt.Errorf("SplitFiles: error reading file: %v", err)
			}
			csvReader := csv.NewReader(csvFile)
			csvHeader, err := csvReader.Read()
			var part int64 = 1
			for {
				newFile, eof, err := WritePart(logger, absPath, csvReader, csvHeader, part, maxBytes)
				if err != nil {
					return nil, fmt.Errorf("SplitFiles: %v", err)
				}
				info, err := os.Stat(newFile)
				if err != nil {
					return nil, fmt.Errorf("SplitFiles: %v", err)
				}
				splitFiles = append(splitFiles, info)
				part++
				if eof || part >= maxSplits {
					break
				}
			}
			os.Remove(absPath)
			fmt.Println(splitFiles)
		} else {
			splitFiles = append(splitFiles, file)
		}
	}
	return splitFiles, nil
}

// MoveFiles moves files from reportsDirectory to stagingDirectory
func MoveFiles(logger logr.Logger, reportsDir, stagingDir dirconfig.Directory) error {
	log := logger.WithValues("costmanagement", "Split")
	// remove all files from directory
	log.Info("Clearing out staging directory!")
	if err := os.RemoveAll(stagingDir.Path); err != nil {
		return fmt.Errorf("MoveFiles: could not clear stagings: %v", err)
	}

	// check if reports/staging exist and recreate if necessary
	if err := dirconfig.ExistsOrRecreate(log, reportsDir, stagingDir); err != nil {
		return fmt.Errorf("MoveFiles: could not check directories")
	}

	// move all files
	fileList, err := ioutil.ReadDir(reportsDir.Path)
	if err != nil {
		return fmt.Errorf("MoveFiles: could not read reports directory: %v", err)
	}
	if len(fileList) <= 0 {
		return ErrNoReports
	}

	log.Info("Moving report files to staging directory")
	for _, file := range fileList {
		if strings.Contains(file.Name(), ".csv") {
			if err := os.Rename(path.Join(reportsDir.Path, file.Name()), path.Join(stagingDir.Path, file.Name())); err != nil {
				return fmt.Errorf("MoveFiles: failed to move files: %v", err)
			}
		}
	}
	return nil
}

// Split is responsible for packing the files for upload
func Split(logger logr.Logger, dirCfg *dirconfig.DirectoryConfig, cost *costmgmtv1alpha1.CostManagement, maxSize int64) ([]os.FileInfo, error) {
	log := logger.WithValues("costmanagement", "Split")
	maxBytes := maxSize * megaByte
	var outFiles []os.FileInfo

	// move CSV reports from data directory to staging directory
	if err := MoveFiles(logger, dirCfg.Reports, dirCfg.Staging); err != nil {
		return nil, err
	}
	// create the upload directory if it does not exist
	if err := dirconfig.ExistsOrRecreate(log, dirCfg.Upload); err != nil {
		return nil, fmt.Errorf("Split: could not check upload directory")
	}

	// check if the files need to be split
	log.Info("Checking to see if the report files need to be split")
	filesToPackage, needSplit, err := NeedSplit(dirCfg.Staging.Path, maxBytes)
	if err != nil {
		return nil, fmt.Errorf("Split: %v", err)
	}
	if needSplit {
		log.Info("Report files exceed the max size. Splitting files")
		filesToPackage, err := SplitFiles(logger, dirCfg.Staging.Path, filesToPackage, maxBytes)
		if err != nil {
			return nil, fmt.Errorf("Split: %v", err)
		}
		fileList, err := BuildLocalCSVFileList(filesToPackage, dirCfg.Staging.Path)
		if err != nil {
			return nil, fmt.Errorf("Split: %v", err)
		}
		manifestFileName, manifestUUID, err := RenderManifest(logger, fileList, cost, dirCfg.Staging.Path)
		if err != nil {
			return nil, fmt.Errorf("Split: %v", err)
		}
		for idx, fileName := range fileList {
			if strings.Contains(fileName, ".csv") {
				fileList = []string{fileName}
				tarFileName := path.Join(dirCfg.Upload.Path, "cost-mgmt"+strconv.Itoa(idx)+".tar.gz")
				log.Info("Generating tar.gz", "tarFile", tarFileName)
				err := WriteTarball(logger, tarFileName, manifestFileName, manifestUUID, fileList, idx)
				if err == ErrNoReports {
					return nil, ErrNoReports
				} else if err != nil {
					return nil, fmt.Errorf("Split: %v", err)
				}
			}
		}
	} else {
		tarFileName := path.Join(dirCfg.Upload.Path, "cost-mgmt.tar.gz")
		log.Info("Report files do not require split, generating tar.gz", "tarFile", tarFileName)
		fileList, err := BuildLocalCSVFileList(filesToPackage, dirCfg.Staging.Path)
		if err != nil {
			return nil, fmt.Errorf("Split: %v", err)
		}
		if len(fileList) > 0 {
			manifestFileName, manifestUUID, err := RenderManifest(logger, fileList, cost, dirCfg.Staging.Path)
			if err != nil {
				return nil, fmt.Errorf("Split: %v", err)
			}
			if err := WriteTarball(logger, tarFileName, manifestFileName, manifestUUID, fileList); err == ErrNoReports {
				return nil, ErrNoReports
			} else if err != nil {
				return nil, fmt.Errorf("Split: %v", err)
			}
		}
	}

	outFiles, err = ioutil.ReadDir(dirCfg.Upload.Path)
	if err != nil {
		return nil, fmt.Errorf("Could not read upload directory: %v", err)
	}

	return outFiles, nil
}

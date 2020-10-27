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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	uuidv4 "github.com/delaemon/go-uuidv4"
	"github.com/go-logr/logr"
	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
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

// define the manifest template
type Manifest struct {
	UUID      string   `json:"uuid"`
	ClusterID string   `json:"cluster_id"`
	Version   string   `json:"version"`
	Date      string   `json:"date"`
	Files     []string `json:"files"`
}

func BuildLocalCSVFileList(stagingDirectory string) ([]string, error) {
	var csvList []string
	fileList, err := ioutil.ReadDir(stagingDirectory)
	if err != nil {
		fmt.Println("could not read dir")
		return csvList, err
	}
	for _, file := range fileList {
		if strings.Contains(file.Name(), ".csv") {
			csvList = append(csvList, stagingDirectory+"/"+file.Name())
		}
	}
	return csvList, nil
}

func NeedSplit(filepath string, maxBytes int64) (bool, error) {
	var totalSize int64 = 0
	fileList, err := ioutil.ReadDir(filepath)
	if err != nil {
		fmt.Println("could not read dir")
		return false, err
	}
	for _, file := range fileList {
		info, err := os.Stat(filepath + "/" + file.Name())
		if err != nil {
			fmt.Println("could not determine file size")
			return false, err
		}
		fileSize := info.Size()
		totalSize += fileSize
		if fileSize >= maxBytes || totalSize >= maxBytes {
			return true, nil
		}
	}
	return false, nil
}

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
	manifestFileName := filepath + "/manifest.json"
	// write the manifest file
	file, err := json.MarshalIndent(fileManifest, "", " ")
	if err != nil {
		fmt.Println("couldnot marshal manifest file")
		return manifestFileName, manifestUUID, err
	}
	err = ioutil.WriteFile(manifestFileName, file, 0644)
	if err != nil {
		fmt.Println("could not create manifest file")
		return manifestFileName, manifestUUID, err
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

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return err
	}

	return nil
}

func WriteTarball(logger logr.Logger, tarFileName, manifestFileName, manifestUUID string, archiveFiles []string, fileNum ...int) (string, error) {
	index := 0
	if len(fileNum) > 0 {
		index = fileNum[0]
	}
	if len(archiveFiles) <= 0 {
		fmt.Println("no files to add to tar file")
		return "", nil
	}
	// create the tarfile
	tarFile, err := os.Create(tarFileName)
	if err != nil {
		fmt.Println("error creating tar file")
		return "", err
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
				return "", err
			}
		}
	}
	err = addFileToTarWriter(logger, "manifest.json", manifestFileName, tw)
	if err != nil {
		fmt.Println("error creating the tar file")
		return "", err
	}
	return tarFileName, nil

}

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
		if err != nil {
			if err == io.EOF {
				writer.Flush()
				return splitFileName, true, nil
			}
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

func SplitFiles(logger logr.Logger, filePath string, maxBytes int64) error {
	fileList, err := ioutil.ReadDir(filePath)
	if err != nil {
		fmt.Println("could not read dir")
	}
	for _, file := range fileList {
		absPath := filePath + "/" + file.Name()
		info, err := os.Stat(absPath)
		if err != nil {
			return fmt.Errorf("SplitFiles: error getting fileInfo: %v", err)
		}
		fileSize := info.Size()
		if fileSize >= maxBytes {
			var splitFiles []string
			// var csvHeader string
			// open the file
			csvFile, err := os.Open(absPath)
			if err != nil {
				return fmt.Errorf("SplitFiles: error reading file: %v", err)
			}
			csvReader := csv.NewReader(csvFile)
			csvHeader, err := csvReader.Read()
			var part int64 = 1
			for {
				newFile, eof, err := WritePart(logger, absPath, csvReader, csvHeader, part, maxBytes)
				if err != nil {
					return fmt.Errorf("SplitFiles: %v", err)
				}
				splitFiles = append(splitFiles, newFile)
				part++
				if eof || part >= maxSplits {
					break
				}
			}
			os.Remove(absPath)
			fmt.Println(splitFiles)
		}
	}
	return nil
}

func MoveFiles(logger logr.Logger, reportsDirectory, stagingDirectory string) error {
	log := logger.WithValues("costmanagement", "Split")
	// remove all files from directory
	log.Info("Clearing out staging directory!")
	err := os.RemoveAll(stagingDirectory)
	if err != nil {
		return err
	}
	// recreate Directory
	if _, err := os.Stat(stagingDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(stagingDirectory, os.ModePerm); err != nil {
			return err
		}
	}
	// move all files
	log.Info("Moving report files to staging directory")
	fileList, err := ioutil.ReadDir(reportsDirectory)
	if err != nil {
		fmt.Println("could not read dir")
		return err
	}
	for _, file := range fileList {
		if strings.Contains(file.Name(), ".csv") {
			err := os.Rename(reportsDirectory+"/"+file.Name(), stagingDirectory+"/"+file.Name())
			if err != nil {
				fmt.Println("error moving files")
				return err
			}
		}
	}
	return nil
}

func Split(logger logr.Logger, filePath string, cost *costmgmtv1alpha1.CostManagement, maxSize int64) (string, error) {
	log := logger.WithValues("costmanagement", "Split")
	var maxBytes int64 = maxSize * megaByte
	var outFiles []string
	var reportsDirectory string = filePath + "data"
	var stagingDirectory string = filePath + "staging"
	var uploadDirectory string = filePath + "upload"
	// move CSV reports from data directory to staging directory
	err := MoveFiles(logger, reportsDirectory, stagingDirectory)
	if err != nil {
		return uploadDirectory, fmt.Errorf("Split %v", err)
	}
	// create the upload directory if it does not exist
	if _, err := os.Stat(uploadDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadDirectory, os.ModePerm); err != nil {
			return uploadDirectory, fmt.Errorf("Split %v", err)
		}
	}
	// check if the files need to be split
	log.Info("Checking to see if the report files need to be split")
	needSplit, err := NeedSplit(stagingDirectory, maxBytes)
	if err != nil {
		return uploadDirectory, fmt.Errorf("Split %v", err)
	}
	if needSplit {
		log.Info("Report files exceed the max size. Splitting files")
		if err := SplitFiles(logger, stagingDirectory, maxBytes); err != nil {
			return uploadDirectory, fmt.Errorf("Split: %v", err)
		}
		tarFileTmpl := "/cost-mgmt"
		fileList, err := BuildLocalCSVFileList(stagingDirectory)
		if err != nil {
			return uploadDirectory, fmt.Errorf("Split %v", err)
		}
		manifestFileName, manifestUUID, err := RenderManifest(logger, fileList, cost, stagingDirectory)
		if err != nil {
			return uploadDirectory, fmt.Errorf("Split %v", err)
		}
		for idx, fileName := range fileList {
			if strings.Contains(fileName, ".csv") {
				fileList = []string{fileName}
				tarFileName := uploadDirectory + tarFileTmpl + strconv.Itoa(idx) + ".tar.gz"
				log.Info("Generating tar.gz", "tarFile", tarFileName)
				outputTar, err := WriteTarball(logger, tarFileName, manifestFileName, manifestUUID, fileList, idx)
				if err != nil {
					return uploadDirectory, fmt.Errorf("Split %v", err)
				}
				if outputTar != "" {
					outFiles = append(outFiles, outputTar)
				}
			}
		}
	} else {
		tarFileName := uploadDirectory + "/cost-mgmt.tar.gz"
		log.Info("Report files do not require split, generating tar.gz", "tarFile", tarFileName)
		fileList, err := BuildLocalCSVFileList(stagingDirectory)
		if err != nil {
			return uploadDirectory, fmt.Errorf("Split %v", err)
		}
		if len(fileList) > 0 {
			manifestFileName, manifestUUID, err := RenderManifest(logger, fileList, cost, stagingDirectory)
			if err != nil {
				return uploadDirectory, fmt.Errorf("Split %v", err)
			}
			outputTar, err := WriteTarball(logger, tarFileName, manifestFileName, manifestUUID, fileList)
			if err != nil {
				return uploadDirectory, fmt.Errorf("Split %v", err)
			}
			if outputTar != "" {
				outFiles = append(outFiles, outputTar)
			}
		}
	}
	log.Info("Created the following files for upload: ", "files", outFiles)
	return uploadDirectory, nil
}

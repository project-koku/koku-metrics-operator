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
	"fmt"
	//"github.com/gofrs/uuid"
	"encoding/csv"
	"encoding/json"
	uuidv4 "github.com/delaemon/go-uuidv4"
	"io"
	"io/ioutil"
	"os"
	"strings"
	// "time"
	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
)

// DEFAULT_MAX_SIZE := 100
// MEGABYTE := 1024 * 1024
// TEMPLATE := map[string]string{
// 	"files" : "",
// 	"date" : time.Now().format("2006-01-02 15:04:05")
// 	"uuid": "",
// 	"cluster_id": "",
// 	"version": ""
// }
// the csv module doesn't expose the bytes-offset of the
// underlying file object.
// instead, the script estimates the size of the data as VARIANCE percent larger than a
// naïve string concatenation of the CSV fields to cover the overhead of quoting
// and delimiters. This gets close enough for now.
// VARIANCE := 0.03

// // Flag to use when writing to a file. Changed to "w" by the -o flag.
// FILE_FLAG := "x"

// // if we're creating more than 1k files, something is probably wrong.
// MAX_SPLITS := 1000

type Manifest struct {
	Uuid       string   `json:"uuid"`
	Cluster_id string   `json:"cluster_id"`
	Version    string   `json:"version"`
	Date       string   `json:"date"`
	Files      []string `json:"files"`
}

func BuildLocalCSVFileList(stagingDirectory string) []string {
	var csvList []string
	fileList, err := ioutil.ReadDir(stagingDirectory)
	if err != nil {
		fmt.Println("could not read dir")
		// log.Error(err, "Could not read the directory")
	}
	for _, file := range fileList {
		if strings.Contains(file.Name(), ".csv") {
			csvList = append(csvList, stagingDirectory+"/"+file.Name())
		}
	}
	return csvList
}

func NeedSplit(filepath string) bool {
	var totalSize int64 = 0
	var DEFAULT_MAX_SIZE int64 = 100
	var MEGABYTE int64 = 1024 * 1024
	maxBytes := DEFAULT_MAX_SIZE * MEGABYTE
	fileList, err := ioutil.ReadDir(filepath)
	if err != nil {
		fmt.Println("could not read dir")
		// log.Error(err, "Could not read the directory")
	}
	for _, file := range fileList {
		info, err := os.Stat(filepath + file.Name())
		if err != nil {
			return false
		}
		fileSize := info.Size()
		totalSize := totalSize + fileSize
		if fileSize >= maxBytes || totalSize >= maxBytes {
			return true
		}
	}
	return false
}

func RenderManifest(archiveFiles []string, cost *costmgmtv1alpha1.CostManagement, filepath string) (string, string) {
	// setup the manifest
	manifestUUID, _ := uuidv4.Generate()
	manifestDate := metav1.Now()
	var manifestFiles []string
	for idx, _ := range archiveFiles {
		upload_name := manifestUUID + "_openshift_usage_report." + strconv.Itoa(idx) + ".csv"
		manifestFiles = append(manifestFiles, upload_name)
	}
	// manifest.files = manifestFiles
	fileManifest := Manifest{
		Uuid:       manifestUUID,
		Cluster_id: cost.Status.ClusterID,
		Version:    cost.Status.OperatorCommit,
		Date:       manifestDate.UTC().Format("2006-01-02 15:04:05"),
		Files:      manifestFiles,
	}
	manifestFileName := filepath + "/manifest.json"
	// write the manifest file
	file, _ := json.MarshalIndent(fileManifest, "", " ")
	_ = ioutil.WriteFile(manifestFileName, file, 0644)
	// return the manifest file/uuid
	return manifestFileName, manifestUUID
}

func addFileToTarWriter(uploadName, filePath string, tarWriter *tar.Writer) error {
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

func WriteTarball(tarFileName, manifestFileName, manifestUUID string, archiveFiles []string, fileCount int) string {
	if len(archiveFiles) > 0 {
		// create the tarfile
		tarFile, err := os.Create(tarFileName)
		if err != nil {
			fmt.Println("Error!")
		}
		defer tarFile.Close()

		gzipWriter := gzip.NewWriter(tarFile)
		defer gzipWriter.Close()

		tw := tar.NewWriter(gzipWriter)
		defer tw.Close()

		// add the files to the tarFile
		for idx, fileName := range archiveFiles {
			fmt.Println(fileName)
			if strings.Contains(fileName, ".csv") {
				uploadName := manifestUUID + "_openshift_usage_report." + strconv.Itoa(idx) + ".csv"
				fmt.Println(uploadName)
				err := addFileToTarWriter(uploadName, fileName, tw)
				if err != nil {
					fmt.Println(err)
					return ""
					// return errors.New(fmt.Sprintf("Could not add file '%s', to tarball, got error '%s'", filePath, err.Error()))
				}
			}
		}
		addFileToTarWriter("manifest.json", manifestFileName, tw)

		return tarFileName

	}
	return ""
}

func WritePart(fileName string, csvReader *csv.Reader, num int, size int) (string, bool) {
	fileNamePart := strings.TrimSuffix(fileName, ".csv")
	sizeEstimate := 0
	VARIANCE := 0.03
	splitFileName := fileNamePart + strconv.Itoa(num) + ".csv"
	splitFile, err := os.Create(fileName)
	if err != nil {
		fmt.Println("An error occurred:", err)
	}
	// Create the csv writer
	writer := csv.NewWriter(splitFile)
	// err = writer.WriteRow(csvHeader)
	for {
		row, _ := csvReader.Read()
		writer.Write(row)
		rowLen := len(strings.Join(row, ","))
		rowSize := rowLen + int(float64(rowLen)*VARIANCE)
		sizeEstimate = sizeEstimate + rowSize
		if sizeEstimate >= size {
			return splitFileName, false
		}
	}
	// return splitFileName, true
}

func SplitFiles(filePath string, maxSize int64) {
	fileList, err := ioutil.ReadDir(filePath)
	var MEGABYTE int64 = 1024 * 1024
	var MAX_SPLITS int64 = 1000
	size := maxSize * MEGABYTE
	if err != nil {
		fmt.Println("could not read dir")
		// log.Error(err, "Could not read the directory")
	}
	for _, file := range fileList {
		absPath := filePath + file.Name()
		info, err := os.Stat(absPath)
		if err != nil {
			fmt.Println("error: ", err)
		}
		fileSize := info.Size()
		if fileSize >= size {
			var splitFiles []string
			// var csvHeader string
			// open the file
			csvFile, err := os.Open(absPath)
			if err != nil {
				fmt.Println("An error occurred ::", err)
			}
			csvReader := csv.NewReader(csvFile)
			// csvHeader := next(csvReader)
			// content, _ := csvReader.ReadAll()
			// part := 1
			var part int64 = 1
			for {
				newFile, eof := WritePart(absPath, csvReader, int(part), int(size))
				splitFiles = append(splitFiles, newFile)
				part = part + 1
				if eof || part >= MAX_SPLITS {
					break
				}
			}
			os.Remove(absPath)
			fmt.Println(splitFiles)
		}
	}
}

func Split(filePath string, cost *costmgmtv1alpha1.CostManagement) {
	var out_files []string
	needSplit := NeedSplit(filePath)
	var max int64 = 100
	if needSplit {
		SplitFiles(filePath, max)
		tarpath := filePath + "/../"
		tarfiletmpl := "cost-mgmt"
		fileList := BuildLocalCSVFileList(filePath)
		manifestFileName, manifestUUID := RenderManifest(fileList, cost, filePath)
		// fileCount := 0
		for idx, filename := range fileList {
			if strings.Contains(filename, ".csv") {
				tarfilename := tarpath + tarfiletmpl + strconv.Itoa(idx) + ".tar.gz"
				outputTar := WriteTarball(tarfilename, manifestFileName, manifestUUID, fileList, len(fileList))
				if outputTar != "" {
					out_files = append(out_files, outputTar)
				}
			}
		}
	} else {
		tarFileName := filePath + "/../cost-mgmt.tar.gz"
		fileList := BuildLocalCSVFileList(filePath)
		fmt.Println("HEYYYYYYOOOOOO I'M HEREEEEEE ")
		if len(fileList) > 0 {
			manifestFileName, manifestUUID := RenderManifest(fileList, cost, filePath)
			fmt.Println("AFTER MANIFEST YEH!")
			outputTar := WriteTarball(tarFileName, manifestFileName, manifestUUID, fileList, len(fileList))
			if outputTar != "" {
				out_files = append(out_files, outputTar)
			}
		}
	}
	for _, fileName := range out_files {
		fmt.Println(fileName)
	}
}

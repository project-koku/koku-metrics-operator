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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	costmgmtv1alpha1 "github.com/project-koku/korekuta-operator-go/api/v1alpha1"
	"github.com/project-koku/korekuta-operator-go/dirconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FilePackager struct {
	Cost     *costmgmtv1alpha1.CostManagement
	DirCfg   *dirconfig.DirectoryConfig
	Log      logr.Logger
	manifest manifestInfo
	uid      string
	MaxSize  int64
	maxBytes int64
}

// Define the global variables
const megaByte int64 = 1024 * 1024

// the csv module doesn't expose the bytes-offset of the
// underlying file object.
// instead, the script estimates the size of the data as VARIANCE percent larger than a
// naïve string concatenation of the CSV fields to cover the overhead of quoting
// and delimiters. This gets close enough for now.
// VARIANCE := 0.03
const variance float64 = 0.03

// if we're creating more than 1k files, something is probably wrong.
var maxSplits int64 = 1000

// ErrNoReports a "no reports" Error type
var ErrNoReports = errors.New("reports not found")

// Manifest interface
type Manifest interface{}

// manifest template
type manifest struct {
	UUID      string    `json:"uuid"`
	ClusterID string    `json:"cluster_id"`
	Version   string    `json:"version"`
	Date      time.Time `json:"date"`
	Files     []string  `json:"files"`
}

type manifestInfo struct {
	manifest Manifest
	filename string
}

// renderManifest writes the manifest
func (m *manifestInfo) renderManifest() error {
	// write the manifest file
	file, err := json.MarshalIndent(m.manifest, "", " ")
	if err != nil {
		return fmt.Errorf("renderManifest: failed to marshal manifest: %v", err)
	}
	if err := ioutil.WriteFile(m.filename, file, 0644); err != nil {
		return fmt.Errorf("renderManifest: failed to write manifest: %v", err)
	}
	return nil
}

// buildLocalCSVFileList gets the list of files in the staging directory
func (p *FilePackager) buildLocalCSVFileList(fileList []os.FileInfo, stagingDirectory string) map[int]string {
	csvList := make(map[int]string)
	for idx, file := range fileList {
		if strings.HasSuffix(file.Name(), ".csv") {
			csvFilePath := filepath.Join(stagingDirectory, file.Name())
			csvList[idx] = csvFilePath
		}
	}
	return csvList
}

func (p *FilePackager) getManifest(archiveFiles map[int]string, filePath string) {
	// setup the manifest
	manifestDate := metav1.Now()
	var manifestFiles []string
	for idx := range archiveFiles {
		uploadName := p.uid + "_openshift_usage_report." + strconv.Itoa(idx) + ".csv"
		manifestFiles = append(manifestFiles, uploadName)
	}
	p.manifest = manifestInfo{
		manifest: manifest{
			UUID:      p.uid,
			ClusterID: p.Cost.Status.ClusterID,
			Version:   p.Cost.Status.OperatorCommit,
			Date:      manifestDate.UTC(),
			Files:     manifestFiles,
		},
		filename: filepath.Join(filePath, "manifest.json"),
	}
}

func (p *FilePackager) addFileToTarWriter(uploadName, filePath string, tarWriter *tar.Writer) error {
	log := p.Log.WithValues("costmanagement", "addFileToTarWriter")
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

// writeTarball packages the files into tar balls
func (p *FilePackager) writeTarball(tarFileName, manifestFileName string, archiveFiles map[int]string) error {

	// create the tarfile
	tarFile, err := os.Create(tarFileName)
	if err != nil {
		return fmt.Errorf("WriteTarball: error creating tar file: %v", err)
	}
	defer tarFile.Close()

	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()
	tw := tar.NewWriter(gzipWriter)
	defer tw.Close()

	// add the files to the tarFile
	for idx, filePath := range archiveFiles {
		if strings.HasSuffix(filePath, ".csv") {
			uploadName := p.uid + "_openshift_usage_report." + strconv.Itoa(idx) + ".csv"
			if err := p.addFileToTarWriter(uploadName, filePath, tw); err != nil {
				return fmt.Errorf("WriteTarball: failed to create tar file: %v", err)
			}
		}
	}
	if err := p.addFileToTarWriter("manifest.json", manifestFileName, tw); err != nil {
		return fmt.Errorf("WriteTarball: failed to create tar file: %v", err)
	}

	return tarFile.Sync()
}

// WritePart writes a portion of a split file into a new file
func (p *FilePackager) writePart(fileName string, csvReader *csv.Reader, csvHeader []string, num int64) (*os.File, bool, error) {
	log := p.Log.WithValues("costmanagement", "WritePart")
	fileNamePart := strings.TrimSuffix(fileName, ".csv")
	sizeEstimate := 0
	splitFileName := fileNamePart + strconv.FormatInt(num, 10) + ".csv"
	log.Info("Creating file ", "file", splitFileName)
	splitFile, err := os.Create(splitFileName)
	if err != nil {
		return nil, false, fmt.Errorf("WritePart: error creating file: %v", err)
	}
	// Create the csv writer
	writer := csv.NewWriter(splitFile)
	// Preserve the header
	if err := writer.Write(csvHeader); err != nil {
		return nil, false, err
	}
	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			writer.Flush()
			return splitFile, true, nil
		} else if err != nil {
			return nil, false, err
		}
		if err := writer.Write(row); err != nil {
			return nil, false, err
		}
		rowLen := len(strings.Join(row, ","))
		rowSize := rowLen + int(float64(rowLen)*variance)
		sizeEstimate += rowSize
		if sizeEstimate >= int(p.maxBytes) {
			writer.Flush()
			return splitFile, false, nil
		}
	}
}

// splitFiles breaks larger files into smaller ones
func (p *FilePackager) splitFiles(filePath string, fileList []os.FileInfo) ([]os.FileInfo, bool, error) {
	log := p.Log.WithValues("costmanagement", "splitFiles")
	if !p.needSplit(fileList) {
		log.Info("Files do not require splitting.")
		return fileList, false, nil
	}
	log.Info("Files require splitting.")
	var splitFiles []os.FileInfo
	for _, file := range fileList {
		absPath := filepath.Join(filePath, file.Name())
		fileSize := file.Size()
		if fileSize >= p.maxBytes {
			// open the file
			csvFile, err := os.Open(absPath)
			if err != nil {
				return nil, false, fmt.Errorf("SplitFiles: error opening file: %v", err)
			}
			csvReader := csv.NewReader(csvFile)
			csvHeader, err := csvReader.Read()
			if err != nil {
				return nil, false, fmt.Errorf("SplitFiles: error reading file: %v", err)
			}
			var part int64 = 1
			for {
				newFile, eof, err := p.writePart(absPath, csvReader, csvHeader, part)
				if err != nil {
					return nil, false, fmt.Errorf("SplitFiles: %v", err)
				}
				info, err := newFile.Stat()
				if err != nil {
					return nil, false, fmt.Errorf("SplitFiles: %v", err)
				}
				splitFiles = append(splitFiles, info)
				part++
				if eof || part >= maxSplits {
					break
				}
			}
			os.Remove(absPath)
		} else {
			splitFiles = append(splitFiles, file)
		}
	}
	return splitFiles, true, nil
}

// needSplit determines if any of the files to be packaged need to be split.
func (p *FilePackager) needSplit(fileList []os.FileInfo) bool {
	var totalSize int64 = 0
	for _, file := range fileList {
		fileSize := file.Size()
		totalSize += fileSize
		if fileSize >= p.maxBytes || totalSize >= p.maxBytes {
			return true
		}
	}
	return false
}

// MoveFiles moves files from reportsDirectory to stagingDirectory
func (p *FilePackager) moveFiles() ([]os.FileInfo, error) {
	log := p.Log.WithValues("costmanagement", "MoveFiles")
	var movedFiles []os.FileInfo

	// move all files
	fileList, err := ioutil.ReadDir(p.DirCfg.Reports.Path)
	if err != nil {
		return nil, fmt.Errorf("MoveFiles: could not read reports directory: %v", err)
	}
	if len(fileList) <= 0 {
		return nil, ErrNoReports
	}

	// remove all files from staging directory
	if p.Cost.Status.Packaging.PackagingError == "" {
		// Only clear the staging directory if previous packaging was successful
		log.Info("Clearing out staging directory!")
		if err := p.DirCfg.Staging.RemoveContents(); err != nil {
			return nil, fmt.Errorf("MoveFiles: could not clear staging: %v", err)
		}
	}

	log.Info("Moving report files to staging directory")
	for _, file := range fileList {
		if !strings.HasSuffix(file.Name(), ".csv") {
			continue
		}
		from := filepath.Join(p.DirCfg.Reports.Path, file.Name())
		to := filepath.Join(p.DirCfg.Staging.Path, p.uid+"-"+file.Name())
		if err := os.Rename(from, to); err != nil {
			return nil, fmt.Errorf("MoveFiles: failed to move files: %v", err)
		}
		newFile, err := os.Stat(to)
		if err != nil {
			return nil, fmt.Errorf("MoveFiles: failed to get new file stats: %v", err)
		}
		movedFiles = append(movedFiles, newFile)
	}
	return movedFiles, nil
}

// ReadUploadDir returns the fileinfo for each file in the upload dir
func (p *FilePackager) ReadUploadDir() ([]string, error) {
	outFiles, err := ioutil.ReadDir(p.DirCfg.Upload.Path)
	if err != nil {
		return nil, fmt.Errorf("Could not read upload directory: %v", err)
	}
	fileList := []string{}
	for _, file := range outFiles {
		fileList = append(fileList, file.Name())
	}
	return fileList, nil
}

// PackageReports is responsible for packing report files for upload
func (p *FilePackager) PackageReports() error {
	log := p.Log.WithValues("costmanagement", "PackageReports")
	p.maxBytes = p.MaxSize * megaByte
	p.uid = uuid.New().String()

	// create reports/staging/upload directories if they do not exist
	if err := dirconfig.CheckExistsOrRecreate(log, p.DirCfg.Reports, p.DirCfg.Staging, p.DirCfg.Upload); err != nil {
		return fmt.Errorf("PackageReports: could not check directory: %v", err)
	}

	// move CSV reports from data directory to staging directory
	filesToPackage, err := p.moveFiles()
	if err == ErrNoReports {
		return nil
	} else if err != nil {
		return fmt.Errorf("PackageReports: %v", err)
	}

	// check if the files need to be split
	log.Info("Checking to see if the report files need to be split")
	filesToPackage, split, err := p.splitFiles(p.DirCfg.Staging.Path, filesToPackage)
	if err != nil {
		return fmt.Errorf("PackageReports: %v", err)
	}
	fileList := p.buildLocalCSVFileList(filesToPackage, p.DirCfg.Staging.Path)
	p.getManifest(fileList, p.DirCfg.Staging.Path)
	log.Info("Rendering manifest.", "manifest", p.manifest.filename)
	if err := p.manifest.renderManifest(); err != nil {
		return fmt.Errorf("PackageReports: %v", err)
	}

	if split {
		for idx, fileName := range fileList {
			if !strings.HasSuffix(fileName, ".csv") {
				continue
			}
			fileList = map[int]string{idx: fileName}
			tarFileName := "cost-mgmt-" + p.uid + "-" + strconv.Itoa(idx) + ".tar.gz"
			tarFilePath := filepath.Join(p.DirCfg.Upload.Path, tarFileName)
			log.Info("Generating tar.gz", "tarFile", tarFilePath)
			if err := p.writeTarball(tarFilePath, p.manifest.filename, fileList); err != nil {
				return fmt.Errorf("PackageReports: %v", err)
			}
		}
	} else {
		tarFileName := "cost-mgmt-" + p.uid + ".tar.gz"
		tarFilePath := filepath.Join(p.DirCfg.Upload.Path, tarFileName)
		log.Info("Generating tar.gz", "tarFile", tarFilePath)
		if err := p.writeTarball(tarFilePath, p.manifest.filename, fileList); err != nil {
			return fmt.Errorf("PackageReports: %v", err)
		}
	}

	log.Info("File packaging was successful.")
	return nil
}

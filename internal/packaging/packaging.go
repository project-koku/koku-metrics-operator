//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package packaging

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/internal/dirconfig"
	"github.com/project-koku/koku-metrics-operator/internal/strset"
)

// FilePackager struct for defining the packaging vars
type FilePackager struct {
	DirCfg           *dirconfig.DirectoryConfig
	FilesAction      FilesAction
	manifest         manifestInfo
	uid              string
	createdTimestamp string
	maxBytes         int64
	start            time.Time
	end              time.Time
}

type FilesAction func(src, dst string) error

const (
	timestampFormat = "20060102T150405.999999"

	megaByte int64 = 1024 * 1024

	// the csv module does not expose the bytes-offset of the
	// underlying file object.
	// instead, the script estimates the size of the data as VARIANCE percent larger than a
	// naïve string concatenation of the CSV fields to cover the overhead of quoting
	// and delimiters. This gets close enough for now.
	// VARIANCE := 0.03
	variance float64 = 0.03
)

var (
	MoveFiles FilesAction = os.Rename
	CopyFiles FilesAction = copyFile

	// if we're creating more than 1k files, something is probably wrong.
	maxSplits int64 = 1000

	BUFFERSIZE int64 = 10 * megaByte

	// ErrNoReports a "no reports" Error type
	ErrNoReports = errors.New("reports not found")

	// Set boolean on whether community or certified
	isCertified bool = false

	log = logr.Log.WithName("packaging")
)

// Manifest interface
type Manifest interface{}

// manifest template
type manifest struct {
	UUID         string                                `json:"uuid"`
	ClusterID    string                                `json:"cluster_id"`
	Version      string                                `json:"version"`
	Date         time.Time                             `json:"date"`
	Files        []string                              `json:"files"`
	ROSFiles     []string                              `json:"resource_optimization_files"`
	Start        time.Time                             `json:"start"`
	End          time.Time                             `json:"end"`
	CRStatus     metricscfgv1beta1.MetricsConfigStatus `json:"cr_status"`
	Certified    bool                                  `json:"certified"`
	DailyReports bool                                  `json:"daily_reports"`
}

type FileInfoManifest manifest
type manifestInfo struct {
	manifest Manifest
	filename string
}
type fileTracker struct {
	costfiles map[int]string
	rosfiles  map[int]string
	allfiles  map[int]string
}

func newFileTracker() fileTracker {
	return fileTracker{
		costfiles: make(map[int]string, 4),
		rosfiles:  make(map[int]string, 1),
		allfiles:  make(map[int]string, 5),
	}
}

// renderManifest writes the manifest
func (m *manifestInfo) renderManifest() error {
	// write the manifest file
	file, err := json.MarshalIndent(m.manifest, "", " ")
	if err != nil {
		return fmt.Errorf("renderManifest: failed to marshal manifest: %v", err)
	}
	if err := os.WriteFile(m.filename, file, 0644); err != nil {
		return fmt.Errorf("renderManifest: failed to write manifest: %v", err)
	}
	return nil
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	_, err = os.Stat(dst)
	if err == nil {
		return fmt.Errorf("file %s already exists", dst)
	}

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	buf := make([]byte, BUFFERSIZE)
	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}
	return err
}

// buildLocalCSVFileList gets the list of files in the staging directory
func (p *FilePackager) buildLocalCSVFileList(fileList []os.FileInfo, stagingDirectory string) fileTracker {
	tracker := newFileTracker()
	for idx, file := range fileList {
		if !strings.HasSuffix(file.Name(), ".csv") {
			continue
		}
		csvFilePath := filepath.Join(stagingDirectory, file.Name())
		if strings.Contains(file.Name(), "ros-openshift") {
			tracker.rosfiles[idx] = csvFilePath
		} else {
			tracker.costfiles[idx] = csvFilePath
		}
		tracker.allfiles[idx] = csvFilePath
	}
	return tracker
}

func (p *FilePackager) getManifest(archiveFiles fileTracker, filePath string, cr *metricscfgv1beta1.MetricsConfig) {
	// setup the manifest
	manifestDate := metav1.Now()
	var costFiles []string
	for idx := range archiveFiles.costfiles {
		uploadName := p.uid + "_openshift_usage_report." + strconv.Itoa(idx) + ".csv"
		costFiles = append(costFiles, uploadName)
	}
	var rosFiles []string
	for idx := range archiveFiles.rosfiles {
		uploadName := p.uid + "_openshift_usage_report." + strconv.Itoa(idx) + ".csv"
		rosFiles = append(rosFiles, uploadName)
	}
	p.manifest = manifestInfo{
		manifest: manifest{
			UUID:         p.uid,
			ClusterID:    cr.Status.ClusterID,
			Version:      cr.Status.OperatorCommit,
			Date:         manifestDate.UTC(),
			Files:        costFiles,
			ROSFiles:     rosFiles,
			Start:        p.start.UTC(),
			End:          p.end.UTC(),
			Certified:    isCertified,
			CRStatus:     cr.Status,
			DailyReports: true,
		},
		filename: filepath.Join(filePath, "manifest.json"),
	}
}

func (p *FilePackager) addFileToTarWriter(uploadName, filePath string, tarWriter *tar.Writer) error {
	log := log.WithName("addFileToTarWriter")
	log.Info("adding file to tar.gz", "file", filePath)
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
	tarFilePath := filepath.Join(p.DirCfg.Upload.Path, tarFileName)
	log.Info("generating tar.gz", "tarFile", tarFilePath)

	// create the tarfile
	tarFile, err := os.Create(tarFilePath)
	if err != nil {
		return fmt.Errorf("writeTarball: error creating tar file: %v", err)
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
				return fmt.Errorf("writeTarball: failed to create tar file: %v", err)
			}
		}
	}
	if err := p.addFileToTarWriter("manifest.json", manifestFileName, tw); err != nil {
		return fmt.Errorf("writeTarball: failed to create tar file: %v", err)
	}

	return tarFile.Sync()
}

// writePart writes a portion of a split file into a new file
func (p *FilePackager) writePart(fileName string, csvReader *csv.Reader, csvHeader []string, num int64) (*os.File, bool, error) {
	log := log.WithName("writePart")
	fileNamePart := strings.TrimSuffix(fileName, ".csv")
	sizeEstimate := 0
	splitFileName := fileNamePart + strconv.FormatInt(num, 10) + ".csv"
	log.Info("creating file ", "file", splitFileName)
	splitFile, err := os.Create(splitFileName)
	if err != nil {
		return nil, false, fmt.Errorf("writePart: error creating file: %v", err)
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

// getIndex returns the index of an element in an array
func getIndex(array []string, val string) (int, error) {
	for index, value := range array {
		if value == val {
			return index, nil
		}
	}
	err := errors.New("could not index the interval time")
	return -1, err
}

// getStartEnd grabs the start and end interval from the csvFile
func (p *FilePackager) getStartEnd(filePath string) error {
	csvFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("getStartEnd: error opening file: %v", err)
	}
	csvReader := csv.NewReader(csvFile)
	csvHeader, err := csvReader.Read()
	if err != nil {
		return fmt.Errorf("getStartEnd: error reading file: %v", err)
	}
	startIndex, err := getIndex(csvHeader, "interval_start")
	if err != nil {
		return fmt.Errorf("getStartEnd: error getting the start index: %v", err)
	}
	endIndex, err := getIndex(csvHeader, "interval_end")
	if err != nil {
		return fmt.Errorf("getStartEnd: error getting the end index: %v", err)
	}
	// grab the first line to get the initial interval start
	firstLine, err := csvReader.Read()
	if err != nil {
		return fmt.Errorf("getStartEnd: error reading file: %v", err)
	}
	startInterval := firstLine[startIndex]
	p.start, _ = time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", startInterval)
	// need to grab the last line in the file to get the last interval end
	allLines, err := csvReader.ReadAll()
	if err != nil {
		return fmt.Errorf("getStartEnd: error reading file: %v", err)
	}
	var lastLine []string
	if len(allLines) > 0 {
		lastLine = allLines[len(allLines)-1]
	} else {
		lastLine = firstLine
	}
	endInterval := lastLine[endIndex]
	p.end, _ = time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", endInterval)
	return nil
}

// splitFiles breaks larger files into smaller ones
func (p *FilePackager) splitFiles(filePath string, fileList []os.FileInfo) ([]os.FileInfo, bool, error) {
	log := log.WithName("splitFiles")
	if !p.needSplit(fileList) {
		log.Info("files do not require splitting")
		return fileList, false, nil
	}
	log.Info("files require splitting")
	var splitFiles []os.FileInfo
	for _, file := range fileList {
		absPath := filepath.Join(filePath, file.Name())
		fileSize := file.Size()
		if fileSize >= p.maxBytes {
			// open the file
			csvFile, err := os.Open(absPath)
			if err != nil {
				return nil, false, fmt.Errorf("splitFiles: error opening file: %v", err)
			}
			csvReader := csv.NewReader(csvFile)
			csvHeader, err := csvReader.Read()
			if err != nil {
				return nil, false, fmt.Errorf("splitFiles: error reading file: %v", err)
			}
			var part int64 = 1
			for {
				newFile, eof, err := p.writePart(absPath, csvReader, csvHeader, part)
				if err != nil {
					return nil, false, fmt.Errorf("splitFiles: error writing part: %v", err)
				}
				info, err := newFile.Stat()
				if err != nil {
					return nil, false, fmt.Errorf("splitFiles: error getting file stats: %v", err)
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

// moveOrCopyFiles moves files from reportsDirectory to stagingDirectory
func (p *FilePackager) moveOrCopyFiles(cr *metricscfgv1beta1.MetricsConfig) ([]os.FileInfo, error) {
	log := log.WithName("moveOrCopyFiles")
	var files []os.FileInfo

	// move all files
	fileList, err := os.ReadDir(p.DirCfg.Reports.Path)
	if err != nil {
		return nil, fmt.Errorf("moveOrCopyFiles: could not read reports directory: %v", err)
	}
	if len(fileList) <= 0 {
		return nil, ErrNoReports
	}

	// remove all files from staging directory
	if cr.Status.Packaging.PackagingError == "" {
		// Only clear the staging directory if previous packaging was successful
		log.Info("clearing out staging directory")
		if err := p.DirCfg.Staging.RemoveContents(); err != nil {
			return nil, fmt.Errorf("moveOrCopyFiles: could not clear staging: %v", err)
		}
	}

	log.Info("moving or copying report files to staging directory")
	for _, file := range fileList {
		if !strings.HasSuffix(file.Name(), ".csv") {
			continue
		}

		from := filepath.Join(p.DirCfg.Reports.Path, file.Name())
		to := filepath.Join(p.DirCfg.Staging.Path, p.uid+"-"+file.Name())

		if err := p.FilesAction(from, to); err != nil {
			return nil, fmt.Errorf("moveOrCopyFiles: failed to copy/move files: %v", err)
		}

		newFile, err := os.Stat(to)
		if err != nil {
			return nil, fmt.Errorf("moveOrCopyFiles: failed to get new file stats: %v", err)
		}
		files = append(files, newFile)
	}
	return files, nil
}

func (p *FilePackager) TrimPackages(cr *metricscfgv1beta1.MetricsConfig) error {
	log := log.WithName("trimPackages")

	packages, err := p.DirCfg.Upload.GetFiles()
	if err != nil {
		return fmt.Errorf("failed to read upload dir: %v", err)
	}

	datetimesSet := strset.NewSet()
	for _, f := range packages {
		if strings.HasSuffix(f, "tar.gz") {
			datetimesSet.Add(strings.Split(f, "-")[0])
		}
	}

	reportCount := int64(datetimesSet.Len())

	if reportCount <= cr.Spec.Packaging.MaxReports {
		log.Info("number of stored reports within limit")
		cr.Status.Packaging.ReportCount = &reportCount
		return nil
	}

	log.Info("max report count reached: removing oldest reports")

	datetimes := []string{}
	for d := range datetimesSet.Range() {
		datetimes = append(datetimes, d)
	}

	sort.Strings(datetimes)
	ind := len(datetimes) - int(cr.Spec.Packaging.MaxReports)
	filesToExclude := datetimes[0:ind]

	for _, pre := range filesToExclude {
		for _, file := range packages {
			if !strings.HasPrefix(file, pre) {
				continue
			}
			log.Info(fmt.Sprintf("removing report: %s", file))
			if err := os.Remove(filepath.Join(p.DirCfg.Upload.Path, file)); err != nil {
				return fmt.Errorf("failed to remove %s: %v", file, err)
			}
		}
	}

	cr.Status.Packaging.ReportCount = &cr.Spec.Packaging.MaxReports
	return nil
}

// PackageReports is responsible for packing report files for upload
func (p *FilePackager) PackageReports(cr *metricscfgv1beta1.MetricsConfig) error {
	log := log.WithName("PackageReports")
	p.maxBytes = *cr.Status.Packaging.MaxSize * megaByte
	p.uid = uuid.New().String()
	p.createdTimestamp = strings.Replace(time.Now().Format(timestampFormat), ".", "_", 1)

	// create reports/staging/upload directories if they do not exist
	if err := dirconfig.CheckExistsOrRecreate(p.DirCfg.Reports, p.DirCfg.Staging, p.DirCfg.Upload); err != nil {
		return fmt.Errorf("PackageReports: could not check directory: %v", err)
	}

	// move CSV reports from data directory to staging directory
	filesToPackage, err := p.moveOrCopyFiles(cr)
	if err == ErrNoReports {
		log.Info("no payload to generate")
		return nil
	} else if err != nil {
		return fmt.Errorf("PackageReports: %v", err)
	}
	// get the start and end dates from the report
	log.Info("getting the start and end intervals for the manifest")
	for _, file := range filesToPackage {
		absPath := filepath.Join(p.DirCfg.Staging.Path, file.Name())
		if strings.Contains(file.Name(), "cm-openshift-pod") || (p.start.IsZero() && strings.Contains(file.Name(), "ros-openshift")) {
			if err := p.getStartEnd(absPath); err != nil {
				return fmt.Errorf("PackageReports: %v", err)
			}
		}
	}

	// check if the files need to be split
	log.Info("checking to see if the report files need to be split")
	filesToPackage, split, err := p.splitFiles(p.DirCfg.Staging.Path, filesToPackage)
	if err != nil {
		return fmt.Errorf("PackageReports: %v", err)
	}
	tracker := p.buildLocalCSVFileList(filesToPackage, p.DirCfg.Staging.Path)
	p.getManifest(tracker, p.DirCfg.Staging.Path, cr)
	log.Info("rendering manifest", "manifest", p.manifest.filename)
	if err := p.manifest.renderManifest(); err != nil {
		return fmt.Errorf("PackageReports: %v", err)
	}

	filenameBase := p.createdTimestamp + "-cost-mgmt"

	if split {
		for idx, fileName := range tracker.allfiles {
			fileMap := map[int]string{idx: fileName}
			tarFileName := filenameBase + "-" + strconv.Itoa(idx) + ".tar.gz"
			if err := p.writeTarball(tarFileName, p.manifest.filename, fileMap); err != nil {
				return fmt.Errorf("PackageReports: %v", err)
			}
		}
	} else {
		tarFileName := filenameBase + ".tar.gz"
		if err := p.writeTarball(tarFileName, p.manifest.filename, tracker.allfiles); err != nil {
			return fmt.Errorf("PackageReports: %v", err)
		}
	}

	log.Info("file packaging was successful")
	cr.Status.Packaging.LastSuccessfulPackagingTime = metav1.Now()
	return nil
}

func (p *FilePackager) GetFileInfo(file string) (FileInfoManifest, error) {
	log := log.WithName("getFileInfo")
	fileInfo := FileInfoManifest{}
	openFile, err := os.Open(file)
	if err != nil {
		log.Info("Could not open tar.gz")
		return fileInfo, err
	}
	archive, err := gzip.NewReader(openFile)
	if err != nil {
		log.Info("Could not read the tar.gz")
		return fileInfo, err
	}
	tr := tar.NewReader(archive)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fileInfo, err
		}
		if hdr.Name != "manifest.json" {
			continue
		}
		//Using a bytes buffer is an important part to print the values as a string
		buff := new(bytes.Buffer)
		_, err = buff.ReadFrom(tr)
		if err != nil {
			log.Info("Could not read from the buffer.")
			return fileInfo, err
		}
		s := buff.String()
		if err := json.Unmarshal([]byte(s), &fileInfo); err != nil {
			log.Info("Could not load the manifest json.")
			return fileInfo, err
		}
		break // exit `for` loop after processing the manifest.json
	}
	return fileInfo, nil
}

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

package collector

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/project-koku/korekuta-operator-go/strset"
)

type report interface {
	writeReport() error
	getOrCreateFile() (*os.File, bool, error)
	writeToFile(*os.File, bool) error
}

type reportFile struct {
	name      string
	path      string
	size      int64
	queryType string
	queryData mappedCSVStruct
	headers   []string
	rowPrefix string
}

func (r *reportFile) writeReport() error {
	csvFile, fileCreated, err := r.getOrCreateFile()
	if err != nil {
		return fmt.Errorf("failed to get or create %s csv: %v", r.queryType, err)
	}
	defer csvFile.Close()
	// logMsg := fmt.Sprintf("writing %s results to file", r.queryType)
	// logger.WithValues("costmanagement", "writeResults").Info(logMsg, "filename", csvFile.Name(), "data set", r.queryType)
	if err := r.writeToFile(csvFile, fileCreated); err != nil {
		return fmt.Errorf("writeReport: %v", err)
	}
	fileInfo, err := csvFile.Stat()
	if err != nil {
		return fmt.Errorf("writeReport: %v", err)
	}
	r.size = fileInfo.Size()
	return nil
}

func (r *reportFile) getOrCreateFile() (*os.File, bool, error) {
	if _, err := os.Stat(r.path); os.IsNotExist(err) {
		if err := os.MkdirAll(r.path, os.ModePerm); err != nil {
			return nil, false, err
		}
	}
	filePath := filepath.Join(r.path, r.name)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		file, err := os.Create(filePath)
		return file, true, err
	}
	if err != nil {
		return nil, false, err
	}
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_RDWR, 0644)
	return file, false, err
}

// writeToFile compares the data to what is in the file and only adds new data to the file
func (r *reportFile) writeToFile(file *os.File, created bool) error {
	set, err := readCSVByLine(file, strset.NewSet(), r.rowPrefix)
	if err != nil {
		return fmt.Errorf("writeToFile: failed to read csv: %v", err)
	}
	cw := csv.NewWriter(file)
	if created {
		if err := cw.Write(r.headers); err != nil {
			return fmt.Errorf("writeToFile: %v", err)
		}
	}

	for _, row := range r.queryData {
		if !set.Contains(row.String()) {
			if err := cw.Write(row.CSVrow()); err != nil {
				return err
			}
		}
	}

	cw.Flush()
	return file.Sync()
}

// readCsv reads the file and puts each row into a set
func readCsv(f *os.File, set *strset.Set) (*strset.Set, error) {
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		set.Add(scanner.Text())
	}
	return set, scanner.Err()
}

// readCSVByLine reads the file and puts each row into a set
func readCSVByLine(f *os.File, set *strset.Set, prefix string) (*strset.Set, error) {
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			set.Add(scanner.Text())
		}
	}
	return set, scanner.Err()
}

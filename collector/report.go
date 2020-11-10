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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/project-koku/korekuta-operator-go/strset"
)

type report interface {
	writeReport() error
	getOrCreateFile() (*os.File, bool, error)
	writeToFile(io.Writer, *strset.Set, bool) error
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
	set, err := readCSV(csvFile, strset.NewSet(), r.rowPrefix)
	if err != nil {
		return fmt.Errorf("writeToFile: failed to read csv: %v", err)
	}
	if err := r.writeToFile(csvFile, set, fileCreated); err != nil {
		return fmt.Errorf("writeReport: %v", err)
	}
	fileInfo, err := csvFile.Stat()
	if err != nil {
		return fmt.Errorf("writeReport: %v", err)
	}
	r.size = fileInfo.Size()
	return csvFile.Sync()
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

// writeToFile writes the rows to file. Writes headers if file was created.
func (r *reportFile) writeToFile(file io.Writer, set *strset.Set, created bool) error {
	cw := csv.NewWriter(file)
	if created {
		if err := cw.Write(r.headers); err != nil {
			return fmt.Errorf("writeToFile: failed to write headers: %v", err)
		}
	}

	for _, row := range r.queryData {
		if !set.Contains(row.string()) {
			if err := cw.Write(row.csvRow()); err != nil {
				return fmt.Errorf("writeToFile: failed to write data row: %v", err)
			}
		}
	}

	cw.Flush()
	return nil
}

// readCSV reads the file and puts each row into a set, excluding rows that do not start with prefix.
func readCSV(handle io.Reader, set *strset.Set, prefix string) (*strset.Set, error) {
	scanner := bufio.NewScanner(handle)
	scanner.Scan() // skip headers
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			set.Add(scanner.Text())
		}
	}
	return set, scanner.Err()
}

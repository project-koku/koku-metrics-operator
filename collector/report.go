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

type dataInterface interface {
	writeToFile(io.Writer, *strset.Set, bool) error
	getPrefix() string
}

type fileInterface interface {
	getName() string
	getOrCreateFile() (*os.File, bool, error)
}

type data struct {
	queryData mappedCSVStruct
	headers   []string
	prefix    string
}

type file struct {
	name string
	path string
}

type report struct {
	data dataInterface
	file fileInterface
	size int64
}

// writeToFile writes the rows to file. Writes headers if file was created.
func (d *data) writeToFile(file io.Writer, set *strset.Set, created bool) error {
	cw := csv.NewWriter(file)
	if created {
		if err := cw.Write(d.headers); err != nil {
			return fmt.Errorf("writeToFile: failed to write headers: %v", err)
		}
	}
	for _, row := range d.queryData {
		if !set.Contains(row.string()) {
			if err := cw.Write(row.csvRow()); err != nil {
				return fmt.Errorf("writeToFile: failed to write data row: %v", err)
			}
		}
	}
	cw.Flush()
	return nil
}

func (d *data) getPrefix() string {
	return d.prefix
}

func (f *file) getName() string {
	return f.name
}

func (f *file) getOrCreateFile() (*os.File, bool, error) {
	if _, err := os.Stat(f.path); os.IsNotExist(err) {
		if err := os.MkdirAll(f.path, os.ModePerm); err != nil {
			return nil, false, err
		}
	}
	filePath := filepath.Join(f.path, f.name)
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

func (r *report) writeReport() error {
	csvFile, fileCreated, err := r.file.getOrCreateFile()
	if err != nil {
		return fmt.Errorf("writeReport: failed to get or create csv: %v", err)
	}
	defer csvFile.Close()
	set, err := readCSV(csvFile, strset.NewSet(), r.data.getPrefix())
	if err != nil {
		return fmt.Errorf("writeReport: failed to read csv: %v", err)
	}
	if err := r.data.writeToFile(csvFile, set, fileCreated); err != nil {
		return fmt.Errorf("writeReport: failed to write to file: %v", err)
	}
	fileInfo, err := csvFile.Stat()
	if err != nil {
		return fmt.Errorf("writeReport: failed to get file size: %v", err)
	}
	r.size = fileInfo.Size()
	return csvFile.Sync()
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

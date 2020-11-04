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
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/project-koku/korekuta-operator-go/strset"
)

type report struct {
	fileName    string
	filePath    string
	size        int64
	queryType   string
	queryData   mappedCSVStruct
	fileHeaders CSVStruct
	rowPrefix   string
}

func (r report) writeReport() error {
	csvFile, created, err := getOrCreateFile(r.filePath, r.fileName)
	if err != nil {
		return fmt.Errorf("failed to get or create %s csv: %v", r.queryType, err)
	}
	defer csvFile.Close()
	logMsg := fmt.Sprintf("writing %s results to file", r.queryType)
	logger.WithValues("costmanagement", "writeResults").Info(logMsg, "filename", csvFile.Name(), "data set", r.queryType)
	if err := writeToFile(csvFile, r.queryData, r.fileHeaders, created); err != nil {
		return fmt.Errorf("writeReport: %v", err)
	}
	fileInfo, err := csvFile.Stat()
	if err != nil {
		return fmt.Errorf("writeReport: %v", err)
	}
	r.size = fileInfo.Size()
	return nil
}

func getOrCreateFile(path, filename string) (*os.File, bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return nil, false, err
		}
	}
	filePath := filepath.Join(path, filename)
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
func writeToFile(file *os.File, data mappedCSVStruct, headers CSVStruct, created bool) error {
	set, err := readCsv(file, strset.NewSet())
	if err != nil {
		return fmt.Errorf("writeToFile: failed to read csv: %v", err)
	}
	if created {
		if err := headers.CSVheader(file); err != nil {
			return fmt.Errorf("writeToFile: %v", err)
		}
	}

	for _, row := range data {
		if !set.Contains(row.String()) {
			if err := row.CSVrow(file); err != nil {
				return err
			}
		}
	}

	return file.Sync()
}

// readCsv reads the file and puts each row into a set
func readCsv(f *os.File, set *strset.Set) (*strset.Set, error) {
	lines, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return set, err
	}
	for _, line := range lines {
		set.Add(strings.Join(line, ","))
	}
	return set, nil
}

// readCSVByLine reads the file and puts each row into a set
func readCSVByLine(f *os.File, set *strset.Set, prefix string) (*strset.Set, error) {
	reader := csv.NewReader(f)
	for {
		line, err := reader.Read()
		switch {
		case err == io.EOF:
			return set, nil
		case err != nil:
			return nil, err
		}
		stringLine := strings.Join(line, ",")
		if strings.HasPrefix(stringLine, prefix) {
			set.Add(stringLine)
		}
	}
}

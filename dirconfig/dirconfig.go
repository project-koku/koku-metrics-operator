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

package dirconfig

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
)

var (
	parentDir    = "/tmp/koku-metrics-operator-reports/"
	queryDataDir = "data"
	stagingDir   = "staging"
	uploadDir    = "upload"
	// archive      = "archive"
)

type DirListFunc = func(path string) ([]os.FileInfo, error)
type RemoveAllFunc = func(path string) error
type StatFunc = func(path string) (os.FileInfo, error)
type DirCreateFunc = func(path string) error

type DirectoryFileSystem struct {
	ListDirectory   DirListFunc
	RemoveAll       RemoveAllFunc
	Stat            StatFunc
	CreateDirectory DirCreateFunc
}

// DirectoryConfig stores the path for each directory
type DirectoryConfig struct {
	Parent  Directory
	Upload  Directory
	Staging Directory
	Reports Directory
	Archive Directory
	*DirectoryFileSystem
}

type Directory struct {
	Path string
	*DirectoryFileSystem
}

func (dir *Directory) String() string {
	return string(dir.Path)
}

func (dir *Directory) RemoveContents() error {
	listDir := ioutil.ReadDir
	removeAll := os.RemoveAll
	if dir.DirectoryFileSystem != nil {
		listDir = dir.DirectoryFileSystem.ListDirectory
		removeAll = dir.DirectoryFileSystem.RemoveAll
	}

	fileList, err := listDir(dir.Path)
	if err != nil {
		return fmt.Errorf("RemoveContents: could not read directory: %v", err)
	}
	for _, file := range fileList {
		if err := removeAll(filepath.Join(dir.Path, file.Name())); err != nil {
			return fmt.Errorf("RemoveContents: could not remove file: %v", err)
		}
	}
	return nil
}

func (dir *Directory) Exists() bool {
	stat := os.Stat
	if dir.DirectoryFileSystem != nil {
		stat = dir.DirectoryFileSystem.Stat
	}
	_, err := stat(dir.Path)
	switch {
	case os.IsNotExist(err):
		return false
	case err != nil:
		return false
	default:
		return true
	}
}

func (dir *Directory) Create() error {
	dirCreator := func(path string) error {
		return os.MkdirAll(path, os.ModePerm)
	}
	if dir.DirectoryFileSystem != nil {
		dirCreator = dir.DirectoryFileSystem.CreateDirectory
	}
	if err := dirCreator(dir.String()); err != nil {
		return fmt.Errorf("Create: %s: %v", dir, err)
	}
	return nil
}

func CheckExistsOrRecreate(log logr.Logger, dirs ...Directory) error {
	for _, dir := range dirs {
		if !dir.Exists() {
			log.Info(fmt.Sprintf("Recreating %s", dir.Path))
			if err := dir.Create(); err != nil {
				return err
			}
		}
	}
	return nil
}

func getOrCreatePath(directory string, dirFs *DirectoryFileSystem) (*Directory, error) {
	dir := Directory{Path: directory, DirectoryFileSystem: dirFs}
	if dir.Exists() {
		return &dir, nil
	}
	if err := dir.Create(); err != nil {
		return nil, err
	}
	return &dir, nil
}

func (dirCfg *DirectoryConfig) GetDirectoryConfig() error {
	var err error
	dirMap := map[string]*Directory{}
	dirMap["parent"], err = getOrCreatePath(parentDir, dirCfg.DirectoryFileSystem)
	if err != nil {
		return fmt.Errorf("getDirectoryConfig: %v", err)
	}

	folders := map[string]string{
		"reports": queryDataDir,
		"staging": stagingDir,
		"upload":  uploadDir,
	}
	for name, folder := range folders {
		d := filepath.Join(parentDir, folder)
		dirMap[name], err = getOrCreatePath(d, dirCfg.DirectoryFileSystem)
		if err != nil {
			return fmt.Errorf("getDirectoryConfig: %v", err)
		}
	}

	return mapstructure.Decode(dirMap, &dirCfg)
}

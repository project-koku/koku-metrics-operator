//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package dirconfig

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/mitchellh/mapstructure"
	logr "sigs.k8s.io/controller-runtime/pkg/log"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
)

var (
	MountPath = filepath.Join("tmp", fmt.Sprintf("%s-metrics-operator-reports", metricscfgv1beta1.NamePrefix))

	queryDataDir = "data"
	stagingDir   = "staging"
	uploadDir    = "upload"

	log = logr.Log.WithName("dirconfig")
)

type DirListFunc = func(path string) ([]fs.DirEntry, error)
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
	listDir := os.ReadDir
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

func (dir *Directory) GetFiles() ([]string, error) {
	outFiles, err := os.ReadDir(dir.Path)
	if err != nil {
		return nil, fmt.Errorf("could not read upload directory: %v", err)
	}
	fileList := []string{}
	for _, file := range outFiles {
		fileList = append(fileList, file.Name())
	}
	return fileList, nil
}

func (dir *Directory) GetFilesFullPath() ([]string, error) {
	files, err := dir.GetFiles()
	if err != nil {
		return nil, fmt.Errorf("could not get full file paths: %v", err)
	}
	pathsList := []string{}
	for _, f := range files {
		pathsList = append(pathsList, filepath.Join(dir.Path, f))
	}
	return pathsList, nil
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
		return fmt.Errorf("create: %s: %v", dir, err)
	}
	return nil
}

func CheckExistsOrRecreate(dirs ...Directory) error {
	for _, dir := range dirs {
		if !dir.Exists() {
			log.Info(fmt.Sprintf("recreating %s", dir.Path))
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
	dirMap["parent"], err = getOrCreatePath(MountPath, dirCfg.DirectoryFileSystem)
	if err != nil {
		return fmt.Errorf("getDirectoryConfig: %v", err)
	}

	folders := map[string]string{
		"reports": queryDataDir,
		"staging": stagingDir,
		"upload":  uploadDir,
	}
	for name, folder := range folders {
		d := filepath.Join(MountPath, folder)
		dirMap[name], err = getOrCreatePath(d, dirCfg.DirectoryFileSystem)
		if err != nil {
			return fmt.Errorf("getDirectoryConfig: %v", err)
		}
	}

	return mapstructure.Decode(dirMap, &dirCfg)
}

func (dirCfg *DirectoryConfig) CheckConfig() bool {
	// quite verbose, but iterating through struct fields is hard
	if !dirCfg.Parent.Exists() || !dirCfg.Upload.Exists() || !dirCfg.Staging.Exists() || !dirCfg.Reports.Exists() {
		return false
	}
	return true
}

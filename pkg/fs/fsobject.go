/*
Copyright 2020 Daniël Franke

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package fs is an abstraction for needed filesystem operations.
package fs

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	PathKey = "path"
)

var (
	// ErrIsDir communicates that we didn't expect a directory.
	ErrIsDir = errors.New("file is a directory")

	// ErrIsNotDir communicates that we expected a directory.
	ErrIsNotDir = errors.New("file is not directory")

	// ErrDirNotEmpty communicates that the directory isn't empty.
	ErrDirNotEmpty = errors.New("directory not empty")

	// ErrIsNotFile communicates that the operation only works on normal files.
	ErrIsNotFile = errors.New("file is not a normal file")
)

// FilesystemObject is a representation of a filesystem object.
type FilesystemObject struct {
	Path        string    `json:"path"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"mod_time"`
	IsDir       bool      `json:"is_dir"`

	Mode     os.FileMode         `json:"-"`
	Root     bool                `json:"-"`
	Children []*FilesystemObject `json:"-"`

	logger *zap.Logger
	sync.Mutex
	// Ugly hack so we don't have to retype the field all the time.
	pathField zap.Field
}

// NewFSObj create a new FileSystemObject
func NewFSObj(path string, info os.FileInfo, root bool, logger *zap.Logger) (*FilesystemObject, error) {
	pathField := zap.String(PathKey, path)
	fso := FilesystemObject{
		Path:      path,
		Size:      info.Size(),
		ModTime:   info.ModTime(),
		Mode:      info.Mode(),
		Root:      root,
		IsDir:     info.IsDir(),
		Children:  []*FilesystemObject{},
		logger:    logger,
		pathField: pathField,
	}

	if !fso.IsDir && fso.Mode.IsRegular() {
		err := fso.DetectContentType()
		if err != nil {
			logger.Error("couldn't detect content-type", pathField, zap.Error(err))
			return &FilesystemObject{}, fmt.Errorf("couldn't detect content-type for %s: %w", fso.Path, err)
		}
	}

	return &fso, nil
}

// ObjFromPath stats a path and creates a FilesystemObject from it.
func ObjFromPath(path string, root bool, logger *zap.Logger) (*FilesystemObject, error) {
	pathField := zap.String(PathKey, path)
	fileInfo, err := os.Stat(path)
	if err != nil {
		logger.Error("coudn't stat", pathField, zap.Error(err))
		return &FilesystemObject{}, fmt.Errorf("couldn't stat %s: %w", path, err)
	}

	logger.Debug("creating new object", pathField)
	return NewFSObj(path, fileInfo, root, logger)
}

// DetectContentType detects the mimetype of
func (fso *FilesystemObject) DetectContentType() error {
	if fso.IsDir {
		return ErrIsDir
	}
	fso.logger.Debug("detecting content-type", fso.pathField)

	// We only need the first 512 bytes to detect the content
	buf := make([]byte, 512)

	f, err := os.Open(fso.Path)
	if err != nil {
		fso.logger.Error("couldn't open file", fso.pathField, zap.Error(err))
		return err
	}
	defer f.Close()

	_, err = f.Read(buf)
	if err != nil {
		if errors.Is(err, io.EOF) {
			fso.logger.Debug("received EOF in first 512 bytes", fso.pathField)
			fso.ContentType = ""
			return nil
		}
		fso.logger.Error("couldn't read first 512 bytes", fso.pathField, zap.Error(err))
		return err
	}

	fso.ContentType = http.DetectContentType(buf)

	return nil
}

// Scan recursively scans the directory and populates its children.
func (fso *FilesystemObject) Scan() error {
	if !fso.IsDir {
		return ErrIsNotDir
	}
	fso.Lock()
	defer fso.Unlock()

	if fso.Root {
		fso.logger.Info("scanning directory", fso.pathField)
	} else {
		fso.logger.Debug("scanning directory", fso.pathField)
	}

	// Clean up Children.
	fso.Children = []*FilesystemObject{}

	files, err := ioutil.ReadDir(fso.Path)
	if err != nil {
		fso.logger.Error("couldn't read directory", fso.pathField, zap.Error(err))
		return err
	}

	for _, file := range files {
		path := path.Join(fso.Path, file.Name())
		f, err := ObjFromPath(path, false, fso.logger)
		if err != nil {
			// We're skipping over files we can't read.
			// TODO: Handle these better, but for now they don't matter to us.
			if os.IsPermission(errors.Unwrap(err)) {
				fso.logger.Info("skipping file", zap.String(PathKey, path), zap.Error(err))
				continue
			}
			fso.logger.Error("couldn't create new FSO", zap.String(PathKey, path), zap.Error(err))
			return err
		}
		fso.Children = append(fso.Children, f)
		if f.IsDir {
			err = f.Scan()
			if err != nil {
				fso.logger.Error("couldn't scan child", zap.String(PathKey, f.Path), zap.Error(err))
				return err
			}
		}
	}
	return nil
}

// Clean cleans out all empty directories under the FSO.
func (fso *FilesystemObject) Clean() error {
	if !fso.IsDir {
		return ErrIsNotDir
	}
	if fso.Root {
		fso.logger.Info("cleaning up empty directories", fso.pathField)
	} else {
		fso.logger.Debug("cleaning up empty directories", fso.pathField)
	}

	// Populate the entire tree, but only for the root object
	if fso.Root {
		err := fso.Scan()
		if err != nil {
			fso.logger.Error("couldn't scan for cleanup", fso.pathField, zap.Error(err))
			return err
		}
	}

	fso.Lock()
	defer fso.Unlock()

	newChildren := []*FilesystemObject{}
	for _, f := range fso.Children {
		// We're not touching normal files.
		if !f.IsDir {
			newChildren = append(newChildren, f)
			continue
		}
		err := f.Clean()
		if err != nil {
			if errors.Is(err, ErrDirNotEmpty) {
				newChildren = append(newChildren, f)
				continue
			}
			fso.logger.Error("can't clean up child", zap.String(PathKey, f.Path), zap.Error(err))
			return err
		}
	}
	fso.Children = newChildren

	// Don't delete the root.
	if fso.Root {
		return nil
	}

	// If not empty, we're not going to delete.
	if len(fso.Children) > 0 {
		return ErrDirNotEmpty
	}

	// All checks done, delete the directory.
	fso.logger.Info("deleting empty directory", fso.pathField)
	return fso.Delete()
}

func (fso *FilesystemObject) Open() (*os.File, error) {
	if fso.IsDir || !fso.Mode.IsRegular() {
		return nil, ErrIsNotFile
	}
	return os.Open(fso.Path)
}

func (fso *FilesystemObject) Delete() error {
	fso.logger.Info("Deleting file", fso.pathField)
	err := os.Remove(fso.Path)
	if err != nil {
		fso.logger.Error("Failed deleting file", fso.pathField, zap.Error(err))
		return err
	}
	return nil
}

// GetAllFiles gets all files in the children of the FilesystemObject
func (fso *FilesystemObject) GetAllFiles() []*FilesystemObject {
	r := make([]*FilesystemObject, 0)
	for _, f := range fso.Children {
		if f.IsDir {
			r = append(r, f.GetAllFiles()...)
			continue
		}
		if !f.IsDir && f.Mode.IsRegular() && !strings.HasPrefix(path.Base(f.Path), ".") && !strings.HasSuffix(f.Path, "~") {
			r = append(r, f)
			continue
		}
	}
	return r
}

// IsEqual deterimines if the FSO is the same as on disk.
// Just a quick check to see if the checsum needs to be updated.
func (fso *FilesystemObject) IsEqual(path string, size int64, modTime time.Time) bool {
	return fso.Path == path && fso.Size == size && fso.ModTime == modTime
}

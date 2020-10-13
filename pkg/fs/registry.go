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

package fs

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// WebObject wraps a FSO, to add a webpath.
type WebObject struct {
	*FilesystemObject
	// WebPath is where the file is downloadable.
	WebPath string `json:"web_path"`
}

func newWebObject(webPath, diskPath string, fso *FilesystemObject) *WebObject {
	wp := strings.ReplaceAll(fso.Path, diskPath, strings.TrimRight(webPath, "/"))
	return &WebObject{fso, wp}
}

// Registry is a struct that keeps track of what paths we serve.
type Registry struct {
	// pathFSO maps web paths to FSOs.
	pathFSO map[string]*FilesystemObject
	logger  *zap.Logger
}

// NewRegistry returns a new Register instance.
func NewRegistry(logger *zap.Logger) *Registry {
	return &Registry{
		pathFSO: make(map[string]*FilesystemObject),
		logger:  logger,
	}
}

// Register registers a filesystem root and its corresponding URL path.
func (r *Registry) Register(servePath, diskPath string) error {
	fso, err := ObjFromPath(diskPath, true, r.logger)
	if err != nil {
		return err
	}
	r.logger.Info("Registering root", zap.String("diskPath", fso.Path), zap.String("servePath", servePath))
	r.pathFSO[servePath] = fso
	return nil
}

// GetAllFiles simply returns a list of all files of all registered roots.
func (r *Registry) GetAllFiles() ([]*WebObject, error) {
	fmt.Printf("%+v\n", r.pathFSO)
	f := make([]*WebObject, 0)
	for p, fso := range r.pathFSO {
		err := fso.Clean()
		if err != nil {
			return f, err
		}
		for _, l := range fso.GetAllFiles() {
			f = append(f, newWebObject(p, fso.Path, l))
		}
	}
	return f, nil
}

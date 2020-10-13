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

package server

import (
	"errors"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/ainmosni/mediasync-server/pkg/fs"
	"github.com/ainmosni/mediasync-server/pkg/httputil"
	"go.uber.org/zap"
)

type DownloadHandler struct {
	diskPath  string
	servePath string
	logger    *zap.Logger
}

// NewDownloadHandler creates a new DownloadHandler
func NewDownloadHandler(diskPath, servePath string, logger *zap.Logger) *DownloadHandler {
	logger = logger.With(zap.String("serve_path", servePath), zap.String("disk_path", diskPath))
	logger.Info("Starting download handler")
	return &DownloadHandler{
		diskPath:  diskPath,
		servePath: servePath,
		logger:    logger,
	}
}

// ServeHTTP for the DownloadHandler, mostly checks if the file exists, and then
// routes it based on method.
func (dh DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := dh.logger.With(zap.String("path", r.URL.Path), zap.String("method", r.Method))
	logger.Info("Received HTTP request")

	// Check for any directory traversal problems.
	if containsDotDot(r.URL.Path) {
		httputil.ErrResponse(w, errors.New("invalid path"), http.StatusBadRequest)
	}

	diskPath := path.Join(dh.diskPath, strings.TrimPrefix(r.URL.Path, dh.servePath))
	fso, err := fs.ObjFromPath(diskPath, false, dh.logger)

	if err != nil {
		logger.Error("couldn't serve file", zap.Error(err))
		if os.IsNotExist(errors.Unwrap(err)) {
			httputil.ErrResponse(w, errors.New("file not found"), http.StatusNotFound)
			return
		}
		if os.IsPermission(errors.Unwrap(err)) {
			httputil.ErrResponse(w, errors.New("forbidden"), http.StatusForbidden)
			return
		}
		httputil.ErrResponse(w, err, http.StatusInternalServerError)
	}
	if fso.IsDir || !fso.Mode.IsRegular() {
		err := errors.New("not a regular file")
		logger.Error("non-files not supported", zap.Error(err))
		httputil.ErrResponse(w, err, http.StatusBadRequest)
	}

	switch r.Method {
	case "GET", "HEAD":
		logger.Info("Serving file")
		w.Header().Add("X-MediaServer-Checksum", "NOT_IMPLEMENTED")
		http.ServeFile(w, r, fso.Path)
	case "DELETE":
		err := deleteFile(w, fso)
		if err != nil {
			logger.Error("Failed to delete file", zap.Error(err))
		}
	default:
		httputil.ErrResponse(w, errors.New("method not supported"), http.StatusMethodNotAllowed)
	}
}

func deleteFile(w http.ResponseWriter, fso *fs.FilesystemObject) error {
	err := fso.Delete()
	if httputil.ErrResponse(w, err, http.StatusInternalServerError) {
		return err
	}
	return nil
}

func containsDotDot(p string) bool {
	// If .. is not present at all, we can quickly be done.
	if !strings.Contains(p, "..") {
		return false
	}

	// We only consider .. dangerous if it's a directory specifier, e.g. surrounded by slashes
	for _, ent := range strings.FieldsFunc(p, isSlashRune) {
		if ent == ".." {
			return true
		}
	}
	return false
}

func isSlashRune(r rune) bool { return r == '/' || r == '\\' }

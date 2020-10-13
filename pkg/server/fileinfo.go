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
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ainmosni/mediasync-server/pkg/fs"
	"github.com/ainmosni/mediasync-server/pkg/httputil"
	"go.uber.org/zap"
)

type FileInfoHandler struct {
	logger   *zap.Logger
	registry *fs.Registry
}

func NewFileInfoHandler(registry *fs.Registry, logger *zap.Logger) *FileInfoHandler {
	return &FileInfoHandler{
		logger:   logger,
		registry: registry,
	}
}

// ServeHTTP for the FileInfoHandler, which simply serves all the files in the cache.
func (h *FileInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With(zap.String("path", r.URL.Path), zap.String("method", r.Method))
	logger.Info("Received HTTP request")
	switch m := r.Method; m {
	case "GET":
		h.serveFiles(w, logger)
	default:
		httputil.ErrResponse(w, errors.New("method not supported"), http.StatusMethodNotAllowed)
	}
}

func (h *FileInfoHandler) serveFiles(w http.ResponseWriter, logger *zap.Logger) {
	files, err := h.registry.GetAllFiles()
	if httputil.ErrResponse(w, err, http.StatusInternalServerError) {
		logger.Error("Couldn't scan files.", zap.Error(err))
		return
	}
	f, err := json.Marshal(files)
	if httputil.ErrResponse(w, err, http.StatusInternalServerError) {
		logger.Error("couldn't encode to JSON", zap.Error(err))
		return
	}
	httputil.JSONResponse(w, f, http.StatusOK)
}

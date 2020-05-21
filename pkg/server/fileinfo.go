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
	logger *zap.Logger
}

func NewFileInfoHandler(logger *zap.Logger) *FileInfoHandler {
	return &FileInfoHandler{
		logger: logger,
	}
}

// Serves HTTP for the FileInfoHandler, which simply serves all the files in the cache.
func (h *FileInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.With(zap.String("path", r.URL.Path))
	logger.Info("Received HTTP request")
	switch m := r.Method; m {
	case "GET":
		h.serveCachedFiles(w, r, logger)
	default:
		httputil.ErrResponse(w, errors.New("method not supported"), http.StatusMethodNotAllowed)
	}
}

func (h *FileInfoHandler) serveCachedFiles(w http.ResponseWriter, r *http.Request, logger *zap.Logger) {
	files := fs.GetAllFilesFromCache()
	f, err := json.Marshal(files)
	if httputil.ErrResponse(w, err, http.StatusInternalServerError) {
		return
	}
	httputil.JSONResponse(w, f, http.StatusOK)
}

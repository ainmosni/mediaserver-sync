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

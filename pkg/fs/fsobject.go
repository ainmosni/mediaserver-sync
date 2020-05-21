package fs

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	PATH_KEY = "path"
)

var (
	// ErrIsDir communicates that we didn't expect a directory.
	ErrIsDir = errors.New("file is a directory")

	// ErrIsNotDir communicates that we expected a directory.
	ErrIsNotDir = errors.New("file is not directory")

	// ErrDirNotEmpty communicates that the directory isn't empty.
	ErrDirNotEmpty = errors.New("directory not empty")
)

// FilesystemObject is a representation of a filesystem object.
type FilesystemObject struct {
	Path        string    `json:"path"`
	Hash        string    `json:"hash"`
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
	path_field zap.Field
}

// NewFSObj create a new FileSystemObject
func NewFSObj(path string, info os.FileInfo, root bool, logger *zap.Logger) (*FilesystemObject, error) {
	path_field := zap.String(PATH_KEY, path)
	fso := FilesystemObject{
		Path:       path,
		Size:       info.Size(),
		ModTime:    info.ModTime(),
		Mode:       info.Mode(),
		Root:       root,
		IsDir:      info.IsDir(),
		Children:   []*FilesystemObject{},
		logger:     logger,
		path_field: path_field,
	}

	if !fso.IsDir && fso.Mode.IsRegular() {
		err := fso.GenerateSum()
		if err != nil {
			logger.Error("couldn't generate sum", path_field, zap.Error(err))
			return &FilesystemObject{}, fmt.Errorf("couldn't generate sum for %s: %w", fso.Path, err)
		}
	}

	if !fso.IsDir && fso.Mode.IsRegular() {
		err := fso.DetectContentType()
		if err != nil {
			logger.Error("couldn't detect content-type", path_field, zap.Error(err))
			return &FilesystemObject{}, fmt.Errorf("couldn't detect content-type for %s: %w", fso.Path, err)
		}
	}

	return &fso, nil
}

// FSObjFromPath stats a path and creates a FilesystemObject from it.
func FSObjFromPath(path string, root bool, logger *zap.Logger) (*FilesystemObject, error) {
	path_field := zap.String(PATH_KEY, path)
	fileInfo, err := os.Stat(path)
	if err != nil {
		logger.Error("coudn't stat", path_field, zap.Error(err))
		return &FilesystemObject{}, fmt.Errorf("Couldn't stat %s: %w", path, err)
	}
	f, ok := GetFromCache(path)
	if ok && f.IsEqual(path, fileInfo.Size(), fileInfo.ModTime()) {
		logger.Debug("file size/mtime is equal to cache", path_field)
		return f, nil
	}

	logger.Debug("not equal or not cached, creating new object", path_field)
	return NewFSObj(path, fileInfo, root, logger)
}

// GenerateSum generates a SHA-1 sum for the file.
func (fso *FilesystemObject) GenerateSum() error {
	if fso.IsDir {
		return ErrIsDir
	}

	fso.logger.Debug("generating checksum", fso.path_field)

	f, err := os.Open(fso.Path)
	if err != nil {
		fso.logger.Error("couldn't open file", fso.path_field, zap.Error(err))
		return err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		fso.logger.Error("couldn't generate checksum", fso.path_field, zap.Error(err))
		return err
	}

	fso.Hash = fmt.Sprintf("%x", h.Sum(nil))

	return nil
}

// DetectContentType detects the mimetype of
func (fso *FilesystemObject) DetectContentType() error {
	if fso.IsDir {
		return ErrIsDir
	}
	fso.logger.Debug("detecting content-type", fso.path_field)

	// We only need the first 512 bytes to detect the content
	buf := make([]byte, 512)

	f, err := os.Open(fso.Path)
	if err != nil {
		fso.logger.Error("couldn't open file", fso.path_field, zap.Error(err))
		return err
	}
	defer f.Close()

	_, err = f.Read(buf)
	if err != nil {
		if errors.Is(err, io.EOF) {
			fso.logger.Debug("received EOF in first 512 bytes", fso.path_field)
			fso.ContentType = ""
			return nil
		}
		fso.logger.Error("couldn't read first 512 bytes", fso.path_field, zap.Error(err))
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
		fso.logger.Info("scanning directory", fso.path_field)
	} else {
		fso.logger.Debug("scanning directory", fso.path_field)
	}

	// Clean up Children.
	fso.Children = []*FilesystemObject{}

	files, err := ioutil.ReadDir(fso.Path)
	if err != nil {
		fso.logger.Error("couldn't read directory", fso.path_field, zap.Error(err))
		return err
	}

	for _, file := range files {
		path := path.Join(fso.Path, file.Name())
		f, err := NewFSObj(path, file, false, fso.logger)
		if err != nil {
			// We're skipping over files we can't read.
			// TODO: Handle these better, but for now they don't matter to us.
			if os.IsPermission(errors.Unwrap(err)) {
				fso.logger.Info("skipping file", zap.String(PATH_KEY, path), zap.Error(err))
				continue
			}
			fso.logger.Error("couldn't create new FSO", zap.String(PATH_KEY, path), zap.Error(err))
			return err
		}
		fso.Children = append(fso.Children, f)
		if f.IsDir {
			err = f.Scan()
			if err != nil {
				fso.logger.Error("couldn't scan child", zap.String(PATH_KEY, f.Path), zap.Error(err))
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
		fso.logger.Info("cleaning up empty directories", fso.path_field)
	} else {
		fso.logger.Debug("cleaning up empty directories", fso.path_field)
	}

	// Populate the entire tree, but only for the root object
	if fso.Root {
		err := fso.Scan()
		if err != nil {
			fso.logger.Error("couldn't scan for cleanup", fso.path_field, zap.Error(err))
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
			fso.logger.Error("can't clean up child", zap.String(PATH_KEY, f.Path), zap.Error(err))
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
	// return os.Remove(fso.Path)
	fso.logger.Info("deleting empty directory", fso.path_field)
	DeleteCacheFile(fso)
	return nil

}

// UpdateCache updates the filecache to reflect its current state.
func (fso *FilesystemObject) UpdateCache() {
	if fso.Root {
		fso.logger.Debug("starting cache rebuild")
	}
	fso.logger.Debug("updating cache", fso.path_field)

	for _, f := range fso.Children {
		if f.IsDir {
			f.UpdateCache()
		}

		if !f.IsDir && f.Mode.IsRegular() {
			UpdateCacheFile(f)
		}
	}

}

// IsEqual deterimines if the FSO is the same as on disk, mostly a quick check to see if the checsum needs to be updated.
func (fso *FilesystemObject) IsEqual(path string, size int64, modTime time.Time) bool {
	return fso.Path == path && fso.Size == size && fso.ModTime == modTime
}

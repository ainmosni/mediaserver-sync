package fs

import (
	"sync"
)

var (
	cache FileCache
	lock  sync.Mutex
)

// FileCache is a simple map from path to FilesystemObjects, mostly to not have to recompute the checksum.
type FileCache map[string]*FilesystemObject

func InitialiseCache() {
	cache = make(FileCache)
}

// GetCache gets the global in-memory cache.
func GetFromCache(path string) (*FilesystemObject, bool) {
	if cache == nil {
		InitialiseCache()
	}

	v, ok := cache[path]
	return v, ok
}

// UpdateCache updates the cache with the contents from another FileCache object.
func UpdateCache(fc FileCache) {
	if cache == nil {
		InitialiseCache()
	}

	lock.Lock()
	for k, v := range fc {
		cache[k] = v
	}
	lock.Unlock()
}

// UpdateCacheFile updates a single file in the cache, creating a new entry if needed.
func UpdateCacheFile(fso *FilesystemObject) {
	if cache == nil {
		InitialiseCache()
	}

	lock.Lock()
	cache[fso.Path] = fso
	lock.Unlock()
}

// DeleteCacheFile deletes an entry from the cache.
func DeleteCacheFile(fso *FilesystemObject) {
	if cache == nil {
		InitialiseCache()
	}

	lock.Lock()
	if _, ok := cache[fso.Path]; ok {
		delete(cache, fso.Path)
	}
	lock.Unlock()
}

// GetAllFilesFromCache creates a slice with all cached files.
func GetAllFilesFromCache() []*FilesystemObject {
	fso := make([]*FilesystemObject, len(cache))
	c := 0
	for _, v := range cache {
		fso[c] = v
		c++
	}
	return fso
}

package fs

import (
	"go.uber.org/zap"
	"time"
)

const (
	UPDATE_TIME = 10 * time.Minute
)
// FileMonitor manages scanning directories and cleaning up.
type FileMonitor struct {
	fsObject *FilesystemObject
	ticker *time.Ticker
	logger *zap.Logger
	done chan bool
}

// NewMonitor
func NewMonitor(path string, logger *zap.Logger) (*FileMonitor, error) {
	// Monitors should only monitor fileroots.
	fso, err := FSObjFromPath(path, true, logger)
	if err != nil {
		return nil, err
	}
	// Clean implies a scan.
	err = fso.Clean()
	if err != nil {
		return nil, err
	}

	fso.UpdateCache()

	// Intiialise monitor ticker
	ticker := time.NewTicker(UPDATE_TIME)
	return &FileMonitor{
		fsObject: fso,
		ticker: ticker,
		logger: logger,
		done: make(chan bool),
	}, nil
}

// Monitor cleans and rescans the filesystem structure every UPDATE_TIME.
// Best run in a goroutine.
// TODO: Refactor this using iNotify to update on demand.
func (fm *FileMonitor) Monitor()  {
	for {
		select {
		case <-fm.done:
			return
		case <-fm.ticker.C:
			err := fm.fsObject.Clean()
			if err != nil {
				fm.logger.Error("error doing period clean", zap.String("root_path", fm.fsObject.Path))
			}
			fm.fsObject.UpdateCache()
		}
	}
}

// StopMonitor stops the ticker for this directory
func (fm *FileMonitor) StopMonitor()  {
	fm.done <- true
	fm.ticker.Stop()
	fm.logger.Info("filemonitor stopped")
}

package main

import (
	"fmt"

	"github.com/ainmosni/mediasync-server/pkg/fs"

	"github.com/ainmosni/mediasync-server/pkg/server"

	"github.com/ainmosni/mediasync-server/pkg/config"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Errorf("can't initialise logger: %w", err))
	}

	c, err := config.GetConfig()
	if err != nil {
		logger.Fatal("can't get configuration", zap.Error(err))
	}
	s := server.New("0.0.0.0", 4242, logger)
	for _, p := range c.FilePaths {
		fm, err := fs.NewMonitor(p.DiskPath, logger.With(zap.String("root_path", p.DiskPath)))
		if err != nil {
			logger.Fatal("couldn't start monitor")
		}
		go fm.Monitor()
		defer fm.StopMonitor()
	}
	logger.Info("starting server")
	logger.Fatal("stopping server", zap.Error(s.Serve()))

}

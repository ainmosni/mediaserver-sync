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

package main

import (
	"fmt"
	"strings"

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
	r := fs.NewRegistry(logger)
	s.Handle("/fileinfo", server.NewFileInfoHandler(r, logger))
	for _, p := range c.FilePaths {
		servePath := p.ServePath
		if !strings.HasSuffix(p.ServePath, "/") {
			servePath += "/"
		}

		err := r.Register(servePath, p.DiskPath)
		if err != nil {
			logger.Sugar().Fatalf("Couldn't register",
				zap.String("servePath", servePath),
				zap.String("diskPath", p.DiskPath),
				zap.Error(err),
			)
		}
		s.Handle(servePath, server.NewDownloadHandler(p.DiskPath, servePath, logger))
	}
	logger.Info("starting server")
	logger.Fatal("stopping server", zap.Error(s.Serve()))
}

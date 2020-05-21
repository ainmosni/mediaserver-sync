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
	"net"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

type Server struct {
	host   string
	port   int
	logger *zap.Logger
}

func New(host string, port int, logger *zap.Logger) *Server {
	http.Handle("/fileinfo", NewFileInfoHandler(logger))

	return &Server{
		host:   host,
		port:   port,
		logger: logger,
	}
}

func (s *Server) Serve() error {
	return http.ListenAndServe(net.JoinHostPort(s.host, strconv.Itoa(s.port)), nil)
}

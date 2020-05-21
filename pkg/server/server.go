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

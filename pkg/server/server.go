/*
Copyright 2020 The Kubernetes Authors.

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
	"crypto/tls"
	"net/http"

	"k8s.io/kops/cmd/kops-controller/pkg/config"
)

type Server struct {
	opt    *config.Options
	server *http.Server
}

func NewServer(opt *config.Options) (*Server, error) {
	server := &http.Server{
		Addr: opt.Server.Listen,
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS12,
			PreferServerCipherSuites: true,
		},
	}
	return &Server{
		opt:    opt,
		server: server,
	}, nil
}

func (s *Server) Start() error {
	return s.server.ListenAndServeTLS(s.opt.Server.ServerCertificatePath, s.opt.Server.ServerKeyPath)
}

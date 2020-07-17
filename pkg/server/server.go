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
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gorilla/mux"
	"k8s.io/klog"
	"k8s.io/kops/cmd/kops-controller/pkg/config"
	"k8s.io/kops/pkg/apis/nodeup"
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

	s := &Server{
		opt:    opt,
		server: server,
	}
	r := mux.NewRouter()
	r.Handle("/bootstrap", http.HandlerFunc(s.bootstrap))
	server.Handler = recovery(r)

	return s, nil
}

func (s *Server) Start() error {
	return s.server.ListenAndServeTLS(s.opt.Server.ServerCertificatePath, s.opt.Server.ServerKeyPath)
}

func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		klog.Infof("bootstrap %s no body", r.RemoteAddr)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO: authenticate request

	req := &nodeup.BootstrapRequest{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		klog.Infof("bootstrap %s decode err: %v", r.RemoteAddr, err)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf("failed to decode: %v", err)))
		return
	}

	if req.APIVersion != nodeup.BootstrapAPIVersion {
		klog.Infof("bootstrap %s wrong APIVersion", r.RemoteAddr)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("unexpected APIVersion"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp := &nodeup.BootstrapResponse{}
	_ = json.NewEncoder(w).Encode(resp)
	klog.Infof("bootstrap %s success", r.RemoteAddr)
}

// recovery is responsible for ensuring we don't exit on a panic.
func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)

				klog.Errorf("failed to handle request: threw exception: %v: %s", err, debug.Stack())
			}
		}()

		next.ServeHTTP(w, req)
	})
}
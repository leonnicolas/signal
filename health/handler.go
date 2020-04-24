// Copyright 2020 by the contributors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package health

import (
	"encoding/json"
	"net/http"
	"sync"
)

// basicHandler is a basic Handler implementation.
type basicHandler struct {
	http.ServeMux
	checksMutex     sync.RWMutex
	livenessChecks  map[string]Check
	readinessChecks map[string]Check
}

type HandlerOption func(configuration *handlerConfiguration)

type handlerConfiguration struct {
	liveEndpoint  string
	readyEndpoint string
}

func LiveEndpoint(endpoint string) HandlerOption {
	return func(c *handlerConfiguration) {
		c.liveEndpoint = endpoint
	}
}

func ReadyEndpoint(endpoint string) HandlerOption {
	return func(c *handlerConfiguration) {
		c.readyEndpoint = endpoint
	}
}

// NewHandler creates a new basic Handler
func NewHandler(options ...HandlerOption) Handler {
	configuration := &handlerConfiguration{
		liveEndpoint:  "/live",
		readyEndpoint: "/ready",
	}

	for _, option := range options {
		option(configuration)
	}

	h := &basicHandler{
		livenessChecks:  make(map[string]Check),
		readinessChecks: make(map[string]Check),
	}
	h.Handle(configuration.liveEndpoint, http.HandlerFunc(h.LiveEndpoint))
	h.Handle(configuration.readyEndpoint, http.HandlerFunc(h.ReadyEndpoint))
	return h
}

func (s *basicHandler) LiveEndpoint(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, s.livenessChecks)
}

func (s *basicHandler) ReadyEndpoint(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, s.readinessChecks, s.livenessChecks)
}

func (s *basicHandler) AddLivenessCheck(name string, check Check) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.livenessChecks[name] = check
}

func (s *basicHandler) AddReadinessCheck(name string, check Check) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.readinessChecks[name] = check
}

func (s *basicHandler) collectChecks(checks map[string]Check, resultsOut map[string]string, statusOut *int) {
	s.checksMutex.RLock()
	defer s.checksMutex.RUnlock()
	for name, check := range checks {
		if err := check(); err != nil {
			*statusOut = http.StatusServiceUnavailable
			resultsOut[name] = err.Error()
		} else {
			resultsOut[name] = "OK"
		}
	}
}

func (s *basicHandler) handle(w http.ResponseWriter, r *http.Request, checks ...map[string]Check) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	checkResults := make(map[string]string)
	status := http.StatusOK
	for _, checks := range checks {
		s.collectChecks(checks, checkResults, &status)
	}

	// write out the response code and content type header
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	// if ?hide=1, return an empty body. Kubernetes only cares about the
	// HTTP status code, so we won't waste bytes on the full body.
	if r.URL.Query().Get("hide") == "1" {
		w.Write([]byte("{}\n"))
		return
	}

	// otherwise, write the JSON body ignoring any encoding errors (which
	// shouldn't really be possible since we're encoding a map[string]string).
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "    ")
	encoder.Encode(checkResults)
}

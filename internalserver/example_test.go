// Copyright 2021 by the contributors.
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

package internalserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"

	"github.com/metalmatze/signal/healthcheck"
	"github.com/prometheus/client_golang/prometheus"
)

func Example() {
	registry := prometheus.NewRegistry()
	healthchecks := healthcheck.NewMetricsHandler(healthcheck.NewHandler(), registry)

	// Create the new internalserver Handler
	h := NewHandler(
		WithHealthchecks(healthchecks),
		WithPrometheusRegistry(registry),
	)

	// Make a HTTP request against the internalserver Handler
	fmt.Print(dumpRequest(h, http.MethodGet, "/"))

	// Output:
	// HTTP/1.1 200 OK
	// Connection: close
	// Content-Type: text/html; charset=utf-8
	//
	// <html><head><title>Internal</title></head><body>
	// <p><a href='/live'>/live - Exposes liveness checks</a></p>
	// <p><a href='/metrics'>/metrics - Exposes Prometheus metrics</a></p>
	// <p><a href='/ready'>/ready - Exposes readiness checks</a></p>
	// </body></html>
}

func Example_custom_endpoint() {
	registry := prometheus.NewRegistry()

	// Create the new internalserver Handler like normal.
	h := NewHandler(
		WithPrometheusRegistry(registry),
	)

	// Add a custom endpoint to the internal handler also registering it with the index page.
	h.AddEndpoint("/foo", "My other signal to expose internally",
		func(w http.ResponseWriter, r *http.Request) {
		},
	)

	// Make a HTTP request against the internalserver Handler
	fmt.Print(dumpRequest(h, http.MethodGet, "/"))

	// Output:
	// HTTP/1.1 200 OK
	// Connection: close
	// Content-Type: text/html; charset=utf-8
	//
	// <html><head><title>Internal</title></head><body>
	// <p><a href='/foo'>/foo - My other signal to expose internally</a></p>
	// <p><a href='/metrics'>/metrics - Exposes Prometheus metrics</a></p>
	// </body></html>
}

func dumpRequest(handler http.Handler, method string, path string) string {
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	dump, err := httputil.DumpResponse(rr.Result(), true)
	if err != nil {
		panic(err)
	}
	return strings.Replace(string(dump), "\r\n", "\n", -1)
}

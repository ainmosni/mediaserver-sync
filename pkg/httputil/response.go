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

package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// Response sends a generic response.
func Response(w http.ResponseWriter, contentType string, body []byte, statusCode int) {
	w.Header().Add("content-length", strconv.Itoa(len(body)))
	w.Header().Add("content-type", contentType)
	w.WriteHeader(statusCode)
	fmt.Fprint(w, string(body))
}

// JSONResponse sends a JSON response
func JSONResponse(w http.ResponseWriter, body []byte, statusCode int) {
	Response(w, JSON_CONTENT_TYPE, body, statusCode)
}

// ErrResponse sends an error response if err contains one, returns false if not.
func ErrResponse(w http.ResponseWriter, httpErr error, statusCode int) bool {
	if httpErr == nil {
		return false
	}

	output, err := json.Marshal(struct {
		Error string `json:"error"`
	}{
		Error: httpErr.Error(),
	})

	if err != nil {
		output = []byte{}
	}

	JSONResponse(w, output, statusCode)

	return true
}

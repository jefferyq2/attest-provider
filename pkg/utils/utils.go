/*
   Copyright 2024 Docker attest-provider authors

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
package utils

import (
	"encoding/json"
	"net/http"

	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
	"k8s.io/klog/v2"
)

const (
	apiVersion = "externaldata.gatekeeper.sh/v1beta1"
	kind       = "ProviderResponse"
)

func GatekeeperResponse(results *[]externaldata.Item, systemErr string) []byte {
	response := externaldata.ProviderResponse{
		APIVersion: apiVersion,
		Kind:       kind,
		Response: externaldata.Response{
			Idempotent: true, // mutation requires idempotent results
		},
	}

	if results != nil {
		response.Response.Items = *results
	} else {
		response.Response.SystemError = systemErr
	}

	body, err := json.Marshal(response)
	if err != nil {
		klog.ErrorS(err, "unable to marshal response")
		panic(err)
	}
	return body
}

func GatekeeperError(systemErr string) []byte {
	return GatekeeperResponse(nil, systemErr)
}

// sendResponse sends back the response to Gatekeeper.
func SendResponse(results *[]externaldata.Item, systemErr string, w http.ResponseWriter) {
	body := GatekeeperResponse(results, systemErr)
	klog.InfoS("sending response", "response", string(body))

	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(body)
	if err != nil {
		klog.ErrorS(err, "unable to write response")
		return
	}
}

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

// sendResponse sends back the response to Gatekeeper.
func SendResponse(results *[]externaldata.Item, systemErr string, w http.ResponseWriter) {
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

	klog.InfoS("sending response", "response", response)

	body, err := json.Marshal(response)
	if err != nil {
		klog.ErrorS(err, "unable to marshal response")
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(body)
	if err != nil {
		klog.ErrorS(err, "unable to write response")
		return
	}
}

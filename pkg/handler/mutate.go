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
package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"

	"github.com/docker/attest-provider/pkg/utils"
	"github.com/docker/attest/oci"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
	"k8s.io/klog/v2"
)

type mutateHandler struct{}

func NewMutateHandler() (http.Handler, error) {
	return &mutateHandler{}, nil
}

func (h *mutateHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			klog.Error(string(debug.Stack()))
			klog.ErrorS(fmt.Errorf("%v", r), "panic occurred")
		}
	}()

	// read request body
	requestBody, err := io.ReadAll(req.Body)
	if err != nil {
		utils.SendResponse(nil, fmt.Sprintf("unable to read request body: %v", err), w)
		return
	}

	klog.InfoS("received request", "body", requestBody)

	// parse request body
	var providerRequest externaldata.ProviderRequest
	err = json.Unmarshal(requestBody, &providerRequest)
	if err != nil {
		utils.SendResponse(nil, fmt.Sprintf("unable to unmarshal request body: %v", err), w)
		return
	}

	results := make([]externaldata.Item, 0)

	ctx := req.Context()
	opts := oci.WithOptions(ctx, nil)

	for _, key := range providerRequest.Request.Keys {
		output, err := getReferenceWithDigest(key, opts)
		if err != nil {
			utils.SendResponse(nil, err.Error(), w)
			return
		}

		results = append(results, externaldata.Item{
			Key:   key,
			Value: output,
		})
	}
	utils.SendResponse(&results, "", w)
}

func getReferenceWithDigest(imageRef string, opts []remote.Option) (string, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return "", fmt.Errorf("unable to parse reference %s: %v", imageRef, err)
	}

	// if it already contains a digest, just return it as is
	if _, ok := ref.(name.Digest); ok {
		return ref.String(), nil
	}

	// we need to make a request to the registry to get the digest
	desc, err := remote.Head(ref, opts...)
	if err != nil {
		return "", fmt.Errorf("unable to get digest for reference %s: %v", imageRef, err)
	}

	return ref.Name() + "@" + desc.Digest.String(), nil
}

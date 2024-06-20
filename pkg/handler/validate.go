package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime/debug"

	"github.com/docker/attest/pkg/attest"
	"github.com/docker/attest/pkg/oci"
	"github.com/docker/attest/pkg/policy"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
	"github.com/open-policy-agent/gatekeeper-external-data-provider/pkg/utils"
	"k8s.io/klog/v2"
)

type ValidationResult struct {
	Outcome    attest.Outcome      `json:"outcome"`
	Input      *policy.PolicyInput `json:"input"`
	VSA        *intoto.Statement   `json:"vsa"`
	Violations []policy.Violation  `json:"violations"`
}

func Validate() http.Handler {
	return http.HandlerFunc(validate)
}

func validate(w http.ResponseWriter, req *http.Request) {
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

	// create a tuf client
	tufOutputPath := filepath.Join("/tuf_temp", ".docker", "tuf")
	tufClient, err := createTufClient(tufOutputPath)
	if err != nil {
		utils.SendResponse(nil, err.Error(), w)
		return
	}

	for _, key := range providerRequest.Request.Keys {
		platform := "linux/amd64"
		resolver, err := oci.NewRegistryAttestationResolver(key, platform)
		if err != nil {
			utils.SendResponse(nil, err.Error(), w)
			return
		}

		opts := &policy.PolicyOptions{
			TufClient:       tufClient,
			LocalTargetsDir: filepath.Join("/tuf_temp", ".docker", "policy"), // location to store policy files downloaded from TUF
			LocalPolicyDir:  "",                                              // overrides TUF policy for local policy files if set
		}

		ctx := req.Context()
		debug := true
		ctx = policy.WithPolicyEvaluator(ctx, policy.NewRegoEvaluator(debug))
		result, err := attest.Verify(ctx, opts, resolver)
		if err != nil {
			utils.SendResponse(nil, err.Error(), w)
			return
		}

		results = append(results, externaldata.Item{
			Key: key,
			Value: ValidationResult{
				Outcome:    result.Outcome,
				Input:      result.Input,
				VSA:        result.VSA,
				Violations: result.Violations,
			},
		})
	}
	utils.SendResponse(&results, "", w)
}

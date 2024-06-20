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
	"github.com/docker/attest/pkg/tuf"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
	"github.com/open-policy-agent/gatekeeper-external-data-provider/internal/embed"
	"github.com/open-policy-agent/gatekeeper-external-data-provider/pkg/utils"
	"k8s.io/klog/v2"
)

func Handler() http.Handler {
	return http.HandlerFunc(handler)
}

func handler(w http.ResponseWriter, req *http.Request) {
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
	}

	// iterate over all keys
	for _, key := range providerRequest.Request.Keys {
		// create a resolver for remote attestations
		platform := "linux/amd64"
		resolver, err := oci.NewRegistryAttestationResolver(key, platform)
		if err != nil {
			utils.SendResponse(nil, err.Error(), w)
		}

		// configure policy options
		opts := &policy.PolicyOptions{
			TufClient:       tufClient,
			LocalTargetsDir: filepath.Join("/tuf_temp", ".docker", "policy"), // location to store policy files downloaded from TUF
			LocalPolicyDir:  "",                                              // overrides TUF policy for local policy files if set
		}

		// verify attestations
		ctx := req.Context()
		debug := true
		ctx = policy.WithPolicyEvaluator(ctx, policy.NewRegoEvaluator(debug))
		result, err := attest.Verify(ctx, opts, resolver)
		if err != nil {
			utils.SendResponse(nil, err.Error(), w)
		}
		switch result.Outcome {
		case attest.OutcomeSuccess:
			klog.Info("policy passed")
			results = append(results, externaldata.Item{
				Key:   key,
				Value: "admit: true, message: policy passed",
			})
		case attest.OutcomeFailure:
			klog.Info("policy failed")
			results = append(results, externaldata.Item{
				Key:   key,
				Error: "admit: false, error: policy failed",
			})
		case attest.OutcomeNoPolicy:
			klog.Infof("no policy for image")
			results = append(results, externaldata.Item{
				Key:   key,
				Value: "admit: true, message: no policy",
			})
		}
	}
	utils.SendResponse(&results, "", w)
}

func createTufClient(outputPath string) (*tuf.TufClient, error) {
	// using oci tuf metadata and targets
	metadataURI := "registry-1.docker.io/docker/tuf-metadata:latest"
	targetsURI := "registry-1.docker.io/docker/tuf-targets"
	// example using http tuf metadata and targets
	// metadataURI := "https://docker.github.io/tuf-staging/metadata"
	// targetsURI := "https://docker.github.io/tuf-staging/targets"

	return tuf.NewTufClient(embed.StagingRoot, outputPath, metadataURI, targetsURI, tuf.NewVersionChecker())
}

package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"

	"github.com/docker/attest/pkg/attest"
	"github.com/docker/attest/pkg/oci"
	"github.com/docker/attest/pkg/policy"
	"github.com/docker/attest/pkg/tuf"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/open-policy-agent/frameworks/constraint/pkg/externaldata"
	"github.com/open-policy-agent/gatekeeper-external-data-provider/internal/embed"
	"github.com/open-policy-agent/gatekeeper-external-data-provider/pkg/utils"
	"k8s.io/klog/v2"
)

type ValidationResult struct {
	Outcome    attest.Outcome      `json:"outcome"`
	Input      *policy.PolicyInput `json:"input"`
	VSA        *intoto.Statement   `json:"vsa"`
	Violations []policy.Violation  `json:"violations"`
}

type ValidateHandlerOptions struct {
	TUFRoot        string
	TUFOutputPath  string
	TUFMetadataURL string
	TUFTargetsURL  string

	PolicyDir      string
	PolicyCacheDir string
}

type validateHandler struct {
	opts *ValidateHandlerOptions
}

func NewValidateHandler(opts *ValidateHandlerOptions) (http.Handler, error) {
	handler := &validateHandler{opts: opts}

	// a TUF client can only be used once, so we need to create a new one for each request.
	// we create this one up front to ensure that the TUF root is valid and to pre-load the metadata.
	// TODO: this pre-loading works for the root, targets, snapshot, and timestamp roles, but not for delegated roles.
	_, err := handler.createTUFClient()
	if err != nil {
		return nil, err
	}

	klog.Infof("validate handler initialized with %s TUF root", opts.TUFRoot)

	return handler, nil
}

func (h *validateHandler) createTUFClient() (*tuf.TufClient, error) {
	var rootBytes []byte
	switch h.opts.TUFRoot {
	case "dev":
		rootBytes = embed.DevRoot
	case "staging":
		rootBytes = embed.StagingRoot
	default:
		return nil, fmt.Errorf("invalid tuf root: %s", h.opts.TUFRoot)
	}
	return tuf.NewTufClient(rootBytes, h.opts.TUFOutputPath, h.opts.TUFMetadataURL, h.opts.TUFTargetsURL, tuf.NewVersionChecker())
}

func (h *validateHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			klog.Error(string(debug.Stack()))
			klog.ErrorS(fmt.Errorf("%v", r), "panic occurred")
		}
	}()

	ctx := req.Context()
	debug := true
	ctx = policy.WithPolicyEvaluator(ctx, policy.NewRegoEvaluator(debug))

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

	tufClient, err := h.createTUFClient()
	if err != nil {
		utils.SendResponse(nil, fmt.Sprintf("unable to create TUF client: %v", err), w)
		return
	}

	policyOpts := &policy.PolicyOptions{
		TufClient:       tufClient,
		LocalTargetsDir: h.opts.PolicyCacheDir,
		LocalPolicyDir:  h.opts.PolicyDir,
	}

	results := make([]externaldata.Item, 0)
	for _, key := range providerRequest.Request.Keys {
		platform := "linux/amd64"
		src, err := oci.ParseImageSpec(key, oci.WithPlatform(platform))
		if err != nil {
			utils.SendResponse(nil, err.Error(), w)
			return
		}

		result, err := attest.Verify(ctx, src, policyOpts)
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

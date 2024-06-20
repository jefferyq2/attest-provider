package handler

import (
	"github.com/docker/attest/pkg/tuf"
	"github.com/open-policy-agent/gatekeeper-external-data-provider/internal/embed"
)

func createTufClient(outputPath string) (*tuf.TufClient, error) {
	// using oci tuf metadata and targets
	metadataURI := "registry-1.docker.io/docker/tuf-metadata:latest"
	targetsURI := "registry-1.docker.io/docker/tuf-targets"
	// example using http tuf metadata and targets
	// metadataURI := "https://docker.github.io/tuf-staging/metadata"
	// targetsURI := "https://docker.github.io/tuf-staging/targets"

	return tuf.NewTufClient(embed.StagingRoot, outputPath, metadataURI, targetsURI, tuf.NewVersionChecker())
}

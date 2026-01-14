package provider_test

import (
	"testing"

	"github.com/blang/semver"
	"github.com/caliban0/pulumi-cherry-servers/provider"
	"github.com/pulumi/pulumi-go-provider/integration"
)

func newServer(t *testing.T) integration.Server {
	t.Helper()

	prov, err := provider.Provider()
	if err != nil {
		t.Fatalf("failed to build provider: %v", err)
	}

	server, err := integration.NewServer(
		t.Context(),
		provider.Name,
		semver.MustParse("1.0.0"),
		integration.WithProvider(prov),
	)
	if err != nil {
		t.Fatalf("failed to build provider server: %v", err)
	}

	return server
}

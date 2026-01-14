package provider

import (
	"os"

	"github.com/cherryservers/cherrygo/v3"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

// Version is set by the Go linker.
var Version string //nolint:gochecknoglobals // Injecting the version
// from the linker means we don't have to set it
// manually on each release. It's also the way it's done in the pulumi template.

const Name = "pulumi-cherry-servers"

type Config struct {
	Token string `pulumi:"token" provider:"secret"`
}

func (c *Config) Annotate(a infer.Annotator) {
	a.Describe(&c.Token, "Cherry Servers API token.")
}

var _ infer.Annotated = (*Config)(nil)

func getProjectClient(cfg Config) (projectClient, error) {
	if token, ok := os.LookupEnv("CHERRY_AUTH_TOKEN"); ok {
		cfg.Token = token
	}

	client, err := cherrygo.NewClient(cherrygo.WithAuthToken(cfg.Token))
	if err != nil {
		return nil, err
	}

	return client.Projects, nil
}

var _ projectClientFactory = getProjectClient

func Provider() (p.Provider, error) {
	return infer.NewProviderBuilder().
		WithResources(
			infer.Resource(&Project{getProjectClient}),
		).
		WithDisplayName(Name).
		WithNamespace("caliban0").
		WithConfig(infer.Config(&Config{})).
		Build()
}

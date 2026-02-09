package provider

import (
	"context"
	"fmt"
	"os"
	"time"

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

func getProjectClient(ctx context.Context) (ProjectClient, error) {
	cfg := infer.GetConfig[Config](ctx)

	if token, ok := os.LookupEnv("CHERRY_AUTH_TOKEN"); ok {
		cfg.Token = token
	}

	client, err := cherrygo.NewClient(cherrygo.WithAuthToken(cfg.Token))
	if err != nil {
		return nil, err
	}

	return client.Projects, nil
}

func getServerClient(ctx context.Context) (ServerClient, error) {
	cfg := infer.GetConfig[Config](ctx)

	if token, ok := os.LookupEnv("CHERRY_AUTH_TOKEN"); ok {
		cfg.Token = token
	}

	client, err := cherrygo.NewClient(cherrygo.WithAuthToken(cfg.Token))
	if err != nil {
		return nil, err
	}

	return client.Servers, nil
}

func newGetImagesClientFunc() func(ctx context.Context) (ImageClient, error) {
	m := NewSingleFlightMemoizer[[]string]()
	return func(ctx context.Context) (ImageClient, error) {
		cfg := infer.GetConfig[Config](ctx)

		if token, ok := os.LookupEnv("CHERRY_AUTH_TOKEN"); ok {
			cfg.Token = token
		}

		client, err := cherrygo.NewClient(cherrygo.WithAuthToken(cfg.Token))
		if err != nil {
			return nil, err
		}

		return newCachedImageClient(&m, client.Images), nil
	}
}

func getIPClient(cfg Config) (IPClient, error) {
	if token, ok := os.LookupEnv("CHERRY_AUTH_TOKEN"); ok {
		cfg.Token = token
	}

	client, err := cherrygo.NewClient(cherrygo.WithAuthToken(cfg.Token))
	if err != nil {
		return nil, err
	}

	return client.IPAddresses, nil
}

var _ ProjectClientFactory = getProjectClient

func deploymentContext(ctx context.Context) (context.Context, context.CancelFunc) {
	// Set a 30 minute timeout for server deployment.
	const timeout = time.Minute * 30
	return context.WithTimeout(ctx, timeout)
}

func Provider() (p.Provider, error) {
	const pollInterval = time.Second * 10
	durationFunc, err := DurationWithJitter(pollInterval)
	if err != nil {
		return p.Provider{}, fmt.Errorf("failed to create polling interval duration: %w", err)
	}

	return infer.NewProviderBuilder().
		WithResources(
			infer.Resource(&Project{GetClient: getProjectClient, GetLogger: GetLogger}),
			infer.Resource(&IP{getIPClient}),
			infer.Resource(&Server{
				GetClient:              getServerClient,
				GetImageClient:         newGetImagesClientFunc(),
				GetLogger:              GetLogger,
				DeploymentContext:      deploymentContext,
				DeploymentPollInterval: durationFunc,
			}),
		).
		WithDisplayName(Name).
		WithNamespace("caliban0").
		WithConfig(infer.Config(&Config{})).
		Build()
}

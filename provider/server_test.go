package provider_test

// Pulumi's integration test package's resource lifecycle tests don't have dry-run hooks.

import (
	"context"
	"errors"
	"testing"

	"github.com/caliban0/pulumi-cherry-servers/provider"
	"github.com/cherryservers/cherrygo/v3"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/stretchr/testify/assert"
)

var errUnexpectedAPICall error = errors.New("unexpected cherry servers api call")

type fakeServerClient struct {
}

func (c fakeServerClient) Create(req *cherrygo.CreateServer) (cherrygo.Server, *cherrygo.Response, error) {
	return cherrygo.Server{}, nil, errUnexpectedAPICall
}

func (c fakeServerClient) Update(_ int, _ *cherrygo.UpdateServer) (cherrygo.Server, *cherrygo.Response, error) {
	return cherrygo.Server{}, nil, errUnexpectedAPICall
}

func (c fakeServerClient) Get(_ int, _ *cherrygo.GetOptions) (cherrygo.Server, *cherrygo.Response, error) {
	return cherrygo.Server{}, nil, errUnexpectedAPICall
}

func (c fakeServerClient) Delete(_ int) (cherrygo.Server, *cherrygo.Response, error) {
	return cherrygo.Server{}, nil, errUnexpectedAPICall
}

func (c fakeServerClient) Reinstall(_ int, _ *cherrygo.ReinstallServerFields) (cherrygo.Server, *cherrygo.Response, error) {
	return cherrygo.Server{}, nil, errUnexpectedAPICall
}

func newServer(ctx context.Context) provider.Server {
	return provider.Server{
		GetClient: func(_ context.Context) (provider.ServerClient, error) {
			return fakeServerClient{}, nil
		},
		DeploymentContext: func(context.Context) (context.Context, context.CancelFunc) {
			return context.WithCancel(ctx)
		},
	}
}

func defaultServerArgs() provider.ServerArgs {
	return provider.ServerArgs{
		Plan:           "test",
		ProjectID:      1,
		Region:         "test",
		Hostname:       "test",
		Image:          "test",
		SSHKeyIDs:      []int{1},
		ExtraIPIDs:     []string{"1"},
		UserData:       "1",
		Tags:           map[string]string{"1": "1"},
		Spot:           true,
		Cycle:          "1",
		DiscountCode:   "1",
		BlockStorageID: 1,
		BGP:            true,
		AllowReinstall: true,
	}
}

// assertServerPreview asserts that:
// 1. Response output args match input args.
// 2. Response output state that doesn't have corresponding args is set to zero value.
// 3. No API calls have been made (by asserting there's no error).
func assertServerPreview(t *testing.T, args provider.ServerArgs, resp infer.CreateResponse[provider.ServerState], err error) {
	assert.Equal(t, args, resp.Output.ServerArgs)
	assert.Zero(t, resp.ID)
	assert.Zero(t, resp.Output.Pricing)
	assert.Zero(t, resp.Output.IPs)
	assert.NoError(t, err)
}

func TestCreatePreview(t *testing.T) {
	server := newServer(t.Context())
	input := defaultServerArgs()

	resp, err := server.Create(t.Context(), infer.CreateRequest[provider.ServerArgs]{
		Name:   "",
		Inputs: input,
		DryRun: true,
	})

	assertServerPreview(t, input, resp, err)
}

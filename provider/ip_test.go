package provider_test

import (
	"testing"

	"github.com/caliban0/pulumi-cherry-servers/provider"
	"github.com/cherryservers/cherrygo/v3"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/stretchr/testify/assert"
)

type fakeIPClient struct {
}

func (fakeIPClient) List(_ int, _ *cherrygo.GetOptions) ([]cherrygo.IPAddress, *cherrygo.Response, error) {
	panic("not implemented")
}

func (fakeIPClient) Get(_ string, _ *cherrygo.GetOptions) (cherrygo.IPAddress, *cherrygo.Response, error) {
	panic("not implemented")
}

func (fakeIPClient) Create(_ int, _ *cherrygo.CreateIPAddress) (cherrygo.IPAddress, *cherrygo.Response, error) {
	panic("not implemented")
}

func (fakeIPClient) Remove(_ string) (_ *cherrygo.Response, _ error) {
	panic("not implemented")
}

func (fakeIPClient) Update(_ string, _ *cherrygo.UpdateIPAddress) (cherrygo.IPAddress, *cherrygo.Response, error) {
	panic("not implemented")
}

func (fakeIPClient) Assign(_ string, _ *cherrygo.AssignIPAddress) (cherrygo.IPAddress, *cherrygo.Response, error) {
	panic("not implemented")
}

func (fakeIPClient) Unassign(_ string) (*cherrygo.Response, error) {
	panic("not implemented")
}

func fakeIPClientFactory(_ provider.Config) (provider.IPClient, error) {
	return fakeIPClient{}, nil
}

func TestCreateIP(t *testing.T) {
	p := provider.IP{GetClient: fakeIPClientFactory}

	cases := []struct {
		name string
		req  infer.CreateRequest[provider.IPArgs]
		resp infer.CreateResponse[provider.IPState]
	}{
		{
			name: "dry-run",
			req:  infer.CreateRequest[provider.IPArgs]{DryRun: true},
			resp: infer.CreateResponse[provider.IPState]{},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := p.Create(t.Context(), tt.req)
			assert.Equal(t, tt.resp, resp)
			assert.NoError(t, err)
		})
	}
}

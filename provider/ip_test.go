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

func (fakeIPClient) List(projectID int, opts *cherrygo.GetOptions) (_ []cherrygo.IPAddress, _ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}

func (fakeIPClient) Get(ipID string, opts *cherrygo.GetOptions) (_ cherrygo.IPAddress, _ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}

func (fakeIPClient) Create(projectID int, request *cherrygo.CreateIPAddress) (_ cherrygo.IPAddress, _ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}

func (fakeIPClient) Remove(ipID string) (_ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}

func (fakeIPClient) Update(ipID string, request *cherrygo.UpdateIPAddress) (_ cherrygo.IPAddress, _ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}

func (fakeIPClient) Assign(ipID string, request *cherrygo.AssignIPAddress) (_ cherrygo.IPAddress, _ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}

func (fakeIPClient) Unassign(ipID string) (_ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}


func fakeIPClientFactory (cfg provider.Config) (provider.IPClient, error) {
	return fakeIPClient{}, nil
}

func TestCreateIP(t *testing.T) {
	p := provider.IP{GetClient: fakeIPClientFactory}

	cases := []struct {
		name string
		req infer.CreateRequest[provider.IPArgs]
		resp infer.CreateResponse[provider.IPState]
	} {
		{
			name: "dry-run",
			req: infer.CreateRequest[provider.IPArgs]{DryRun: true},
			resp: infer.CreateResponse[provider.IPState]{},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := p.Create(t.Context(), tt.req)
			assert.NoError(t, err)
			assert.Equal(t, tt.resp, resp)
		})
	}

}
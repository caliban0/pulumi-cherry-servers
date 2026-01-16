package provider_test

import (
	"testing"

	"github.com/caliban0/pulumi-cherry-servers/provider"
	"github.com/cherryservers/cherrygo/v3"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/stretchr/testify/assert"
)

type fakeProjectsClient struct {
}

func (c fakeProjectsClient) List(teamID int, opts *cherrygo.GetOptions) (
	_ []cherrygo.Project, _ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}

func (c fakeProjectsClient) Get(projectID int, opts *cherrygo.GetOptions) (
	_ cherrygo.Project, _ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}

func (c fakeProjectsClient) Create(teamID int, request *cherrygo.CreateProject) (
	_ cherrygo.Project, _ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}

func (c fakeProjectsClient) Update(projectID int, request *cherrygo.UpdateProject) (
	_ cherrygo.Project, _ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}

func (c fakeProjectsClient) ListSSHKeys(projectID int, opts *cherrygo.GetOptions) (
	_ []cherrygo.SSHKey, _ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}

func (c fakeProjectsClient) Delete(projectID int) (_ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}

func fakeProjectClientFactory (cfg provider.Config) (provider.ProjectClient, error) {
	return fakeProjectsClient{}, nil
}

func TestCreateProject(t *testing.T) {
	p := provider.Project{GetClient: fakeProjectClientFactory}

	cases := []struct {
		name string
		req infer.CreateRequest[provider.ProjectArgs]
		resp infer.CreateResponse[provider.ProjectState]
	} {
		{
			name: "dry-run",
			req: infer.CreateRequest[provider.ProjectArgs]{DryRun: true},
			resp: infer.CreateResponse[provider.ProjectState]{},
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

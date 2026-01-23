package provider_test

// Unit tests for stuff that's tricky to cover with integration/lifecycle tests.

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/caliban0/pulumi-cherry-servers/provider"
	"github.com/cherryservers/cherrygo/v3"
	prov "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/stretchr/testify/assert"
)

type projectCreateFunc func(teamID int, request *cherrygo.CreateProject) (cherrygo.Project, *cherrygo.Response, error)
type projectDeleteFunc func(projectID int) (*cherrygo.Response, error)
type projectGetFunc func(projectID int, opts *cherrygo.GetOptions) (_ cherrygo.Project, _ *cherrygo.Response, _ error)

var projectCreateOK projectCreateFunc = func(teamID int, request *cherrygo.CreateProject) (cherrygo.Project, *cherrygo.Response, error) {
	return cherrygo.Project{
		ID:   0,
		Name: request.Name,
		Bgp:  cherrygo.ProjectBGP{Enabled: request.Bgp}}, nil, nil
}

type fakeProjectsClient struct {
	createFunc projectCreateFunc
	deleteFunc projectDeleteFunc
	getFunc    projectGetFunc
}

func (c fakeProjectsClient) List(teamID int, opts *cherrygo.GetOptions) (
	_ []cherrygo.Project, _ *cherrygo.Response, _ error) {
	panic("not implemented") // TODO: Implement
}

func (c fakeProjectsClient) Get(projectID int, opts *cherrygo.GetOptions) (
	_ cherrygo.Project, _ *cherrygo.Response, _ error) {
	if c.getFunc == nil {
		panic("no Get callback for fakeProjectsClient")
	}
	return c.getFunc(projectID, opts)
}

func (c fakeProjectsClient) Create(teamID int, request *cherrygo.CreateProject) (
	_ cherrygo.Project, _ *cherrygo.Response, _ error) {
	if c.createFunc == nil {
		panic("no Create callback for fakeProjectsClient")
	}
	return c.createFunc(teamID, request)
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
	if c.deleteFunc == nil {
		panic("no Delete callback for fakeProjectsClient")
	}
	return c.deleteFunc(projectID)
}

type fakeProjectsClientOption func(*fakeProjectsClient)

func withCreateProject(f projectCreateFunc) fakeProjectsClientOption {
	return func(client *fakeProjectsClient) {
		client.createFunc = f
	}
}

func withDeleteProject(f projectDeleteFunc) fakeProjectsClientOption {
	return func(client *fakeProjectsClient) {
		client.deleteFunc = f
	}
}

func withGetProject(f projectGetFunc) fakeProjectsClientOption {
	return func(client *fakeProjectsClient) {
		client.getFunc = f
	}
}

func newFakeProjectsClientFactory(opts ...fakeProjectsClientOption) provider.ProjectClientFactory {
	return func(_ context.Context) (provider.ProjectClient, error) {
		f := fakeProjectsClient{}
		for _, opt := range opts {
			opt(&f)
		}
		return f, nil
	}
}

func TestDeleteProjectNotFound(t *testing.T) {
	clientFactory := newFakeProjectsClientFactory(withDeleteProject(
		func(projectID int) (*cherrygo.Response, error) {
			return &cherrygo.Response{
				Response: &http.Response{StatusCode: http.StatusNotFound},
			}, errors.New("")
		},
	))

	p := provider.Project{GetClient: clientFactory, GetLogger: GetFakeLogger}

	// Check that "not found" is handled gracefully in deletion operation.
	_, err := p.Delete(t.Context(), infer.DeleteRequest[provider.ProjectState]{ID: "0"})
	assert.NoError(t, err)
}

func TestReadProjectNotFound(t *testing.T) {
	clientFactory := newFakeProjectsClientFactory(withGetProject(
		func(projectID int, opts *cherrygo.GetOptions) (cherrygo.Project, *cherrygo.Response, error) {
			return cherrygo.Project{},
				&cherrygo.Response{Response: &http.Response{StatusCode: http.StatusNotFound}},
				errors.New("")
		},
	))

	p := provider.Project{GetClient: clientFactory, GetLogger: GetFakeLogger}

	// Check that "not found" is handled gracefully in read operation.
	_, err := p.Read(t.Context(), infer.ReadRequest[provider.ProjectArgs, provider.ProjectState]{ID: "0"})
	assert.NoError(t, err)
}

func TestDiffProjectRequiresReplace(t *testing.T) {
	p := provider.Project{}

	// Require replacement if team changes.
	resp, err := p.Diff(t.Context(), infer.DiffRequest[provider.ProjectArgs, provider.ProjectState]{
		State: provider.ProjectState{
			ProjectArgs: provider.ProjectArgs{
				Team: 1,
			},
		},
		Inputs: provider.ProjectArgs{
			Team: 2,
		},
	})

	assert.Equal(t, prov.PropertyDiff{Kind: prov.UpdateReplace}, resp.DetailedDiff["team"])
	assert.NoError(t, err)
}

func TestCreateProject(t *testing.T) {
	cases := []struct {
		name          string
		req           infer.CreateRequest[provider.ProjectArgs]
		resp          infer.CreateResponse[provider.ProjectState]
		clientFactory provider.ProjectClientFactory
	}{
		{
			name: "dry-run",
			req: infer.CreateRequest[provider.ProjectArgs]{
				DryRun: true,
				Inputs: provider.ProjectArgs{Name: "test", BGP: true, Team: 1},
			},
			resp: infer.CreateResponse[provider.ProjectState]{
				Output: provider.ProjectState{ProjectArgs: provider.ProjectArgs{
					Name: "test",
					BGP:  true,
					Team: 1,
				}, LocalASN: 0},
			},
			clientFactory: newFakeProjectsClientFactory(withCreateProject(projectCreateOK)),
		},
		{
			name: "ok",
			req: infer.CreateRequest[provider.ProjectArgs]{
				Inputs: provider.ProjectArgs{Name: "test", BGP: true, Team: 1},
			},
			resp: infer.CreateResponse[provider.ProjectState]{
				ID: "0",
				Output: provider.ProjectState{ProjectArgs: provider.ProjectArgs{
					Name: "test",
					BGP:  true,
					Team: 1,
				}, LocalASN: 0},
			},
			clientFactory: newFakeProjectsClientFactory(withCreateProject(projectCreateOK)),
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			p := provider.Project{GetClient: tt.clientFactory, GetLogger: GetFakeLogger}
			resp, err := p.Create(t.Context(), tt.req)
			assert.NoError(t, err)
			assert.Equal(t, tt.resp, resp)
		})
	}

}

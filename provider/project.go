package provider

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cherryservers/cherrygo/v3"
	prov "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

type projectClient interface {
	cherrygo.ProjectsService
}

type projectClientFactory func(cfg Config) (projectClient, error)

type Project struct {
	getClient projectClientFactory
}

func (p *Project) Annotate(a infer.Annotator) {
	a.Describe(&p, "A Cherry Servers project.")
}

type ProjectArgs struct {
	Name string `pulumi:"name"`
	Team int    `pulumi:"team"`
	Bgp  bool   `pulumi:"bgp,optional"`
}

func (p *ProjectArgs) Annotate(a infer.Annotator) {
	a.Describe(&p.Name, "Project name.")
	a.Describe(&p.Team, "ID of the team the project will belong to.")
	a.Describe(&p.Bgp, "Whether BGP should be enabled for the project.")
}

type ProjectBgpState struct {
	Enabled  bool `pulumi:"enabled"`
	LocalASN int  `pulumi:"localASN"`
}

func (p *ProjectBgpState) Annotate(a infer.Annotator) {
	a.Describe(&p.Enabled, "Whether BGP is enabled.")
	a.Describe(&p.LocalASN, "LocalASN assigned to the project.")
}

type ProjectState struct {
	Name string          `pulumi:"name"`
	Bgp  ProjectBgpState `pulumi:"bgp"`
}

func (p *ProjectState) Annotate(a infer.Annotator) {
	a.Describe(&p.Name, "Project name.")
	a.Describe(&p.Bgp, "Project BGP status.")
}

var (
	_ infer.Annotated = (*Project)(nil)
	_ infer.Annotated = (*ProjectArgs)(nil)
	_ infer.Annotated = (*ProjectBgpState)(nil)
	_ infer.Annotated = (*ProjectState)(nil)
)

func (p *Project) Create(ctx context.Context, req infer.CreateRequest[ProjectArgs]) (
	infer.CreateResponse[ProjectState], error) {
	if req.DryRun {
		return infer.CreateResponse[ProjectState]{}, nil
	}

	client, err := p.getClient(infer.GetConfig[Config](ctx))
	if err != nil {
		return infer.CreateResponse[ProjectState]{}, err
	}

	project, _, err := client.Create(req.Inputs.Team, &cherrygo.CreateProject{
		Name: req.Inputs.Name,
		Bgp:  req.Inputs.Bgp,
	})
	if err != nil {
		return infer.CreateResponse[ProjectState]{}, err
	}

	return infer.CreateResponse[ProjectState]{
		ID: strconv.Itoa(project.ID),
		Output: ProjectState{
			Name: project.Name,
			Bgp: ProjectBgpState{
				Enabled: project.Bgp.Enabled, LocalASN: project.Bgp.LocalASN},
		},
	}, nil
}

func (p *Project) Delete(ctx context.Context, req infer.DeleteRequest[ProjectState]) (infer.DeleteResponse, error) {
	client, err := p.getClient(infer.GetConfig[Config](ctx))
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	id, err := strconv.Atoi(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("id not an int: %w", err)
	}

	r, err := client.Delete(id)
	if err != nil && r.StatusCode == http.StatusNotFound {
		prov.GetLogger(ctx).Warningf("project %s already deleted", req.ID)
		err = nil
	}
	return infer.DeleteResponse{}, err
}

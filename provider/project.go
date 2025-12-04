package provider

import (
	"context"
	"strconv"

	"github.com/cherryservers/cherrygo/v3"
	"github.com/pulumi/pulumi-go-provider/infer"
)

type projectClient interface {
	cherrygo.ProjectsService
}

type projectClientFactory func(cfg Config) (projectClient, error)

type Project struct {
	getClient projectClientFactory
}

type ProjectArgs struct {
	Name string `pulumi:"name"`
	Team int    `pulumi:"team"`
	Bgp  bool   `pulumi:"bgp,optional"`
}

type ProjectBgpState struct {
	Enabled  bool `pulumi:"enabled"`
	LocalASN int  `pulumi:"localASN"`
}

type ProjectState struct {
	Name string          `pulumi:"name"`
	Bgp  ProjectBgpState `pulumi:"bgp"`
}

func (p *Project) Create(ctx context.Context, req infer.CreateRequest[ProjectArgs]) (
	infer.CreateResponse[ProjectState], error) {
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

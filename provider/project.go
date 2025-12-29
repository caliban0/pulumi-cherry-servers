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
	Name string `pulumi:"name,optional"`
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
	Team int             `pulumi:"team"`
	Bgp  ProjectBgpState `pulumi:"bgp"`
}

func (p *ProjectState) Annotate(a infer.Annotator) {
	a.Describe(&p.Name, "Project name.")
	a.Describe(&p.Bgp, "Project BGP status.")
}

var (
	_ infer.Annotated                                       = (*Project)(nil)
	_ infer.Annotated                                       = (*ProjectArgs)(nil)
	_ infer.Annotated                                       = (*ProjectBgpState)(nil)
	_ infer.Annotated                                       = (*ProjectState)(nil)
	_ infer.CustomDelete[ProjectState]                      = (*Project)(nil)
	_ infer.CustomCheck[ProjectArgs]                        = (*Project)(nil)
	_ infer.CustomUpdate[ProjectArgs, ProjectState]         = (*Project)(nil)
	_ infer.CustomDiff[ProjectArgs, ProjectState]           = (*Project)(nil)
	_ infer.CustomRead[ProjectArgs, ProjectState]           = (*Project)(nil)
	_ infer.ExplicitDependencies[ProjectArgs, ProjectState] = (*Project)(nil)
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
		ID:     strconv.Itoa(project.ID),
		Output: projectStateFromClientResp(project, req.Inputs.Team),
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

func (p *Project) Check(ctx context.Context, req infer.CheckRequest) (
	infer.CheckResponse[ProjectArgs], error) {
	args, failures, err := infer.DefaultCheck[ProjectArgs](ctx, req.NewInputs)
	if err != nil {
		return infer.CheckResponse[ProjectArgs]{
			Inputs:   args,
			Failures: failures,
		}, err
	}

	args.Name, err = autoname(args.Name, req.Name, req.OldInputs.Get("name"))
	return infer.CheckResponse[ProjectArgs]{
		Inputs:   args,
		Failures: failures,
	}, err
}

func (p *Project) Update(
	ctx context.Context, req infer.UpdateRequest[ProjectArgs, ProjectState]) (
	infer.UpdateResponse[ProjectState], error) {
	if req.DryRun {
		return infer.UpdateResponse[ProjectState]{
			Output: ProjectState{
				Name: req.Inputs.Name,
				Team: req.Inputs.Team,
				Bgp: ProjectBgpState{
					Enabled: req.Inputs.Bgp,
				},
			},
		}, nil
	}

	client, err := p.getClient(infer.GetConfig[Config](ctx))
	if err != nil {
		return infer.UpdateResponse[ProjectState]{}, err
	}

	id, err := strconv.Atoi(req.ID)
	if err != nil {
		return infer.UpdateResponse[ProjectState]{}, err
	}

	project, _, err := client.Update(id, &cherrygo.UpdateProject{
		Name: &req.Inputs.Name,
		Bgp:  &req.Inputs.Bgp,
	})

	return infer.UpdateResponse[ProjectState]{
		Output: projectStateFromClientResp(project, req.Inputs.Team),
	}, err
}

func (p *Project) Diff(
	_ context.Context, req infer.DiffRequest[ProjectArgs, ProjectState]) (
	infer.DiffResponse, error) {
	diff := map[string]prov.PropertyDiff{}

	if req.Inputs.Name != req.State.Name {
		diff["name"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if req.Inputs.Bgp != req.State.Bgp.Enabled {
		diff["bgp"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if req.Inputs.Team != req.State.Team {
		diff["team"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	return infer.DiffResponse{
		DeleteBeforeReplace: true,
		HasChanges:          len(diff) > 0,
		DetailedDiff:        diff,
	}, nil
}

func (p *Project) Read(
	ctx context.Context, req infer.ReadRequest[ProjectArgs, ProjectState]) (
	infer.ReadResponse[ProjectArgs, ProjectState], error) {
	client, err := p.getClient(infer.GetConfig[Config](ctx))
	if err != nil {
		return infer.ReadResponse[ProjectArgs, ProjectState]{}, err
	}

	id, err := strconv.Atoi(req.ID)
	if err != nil {
		return infer.ReadResponse[ProjectArgs, ProjectState]{}, err
	}

	project, _, err := client.Get(id, nil)
	return infer.ReadResponse[ProjectArgs, ProjectState]{
		ID: req.ID,
		Inputs: ProjectArgs{
			Name: req.Inputs.Name,
			Bgp:  req.Inputs.Bgp,
			Team: req.Inputs.Team,
		},
		State: projectStateFromClientResp(project, req.Inputs.Team),
	}, err
}

func projectStateFromClientResp(p cherrygo.Project, teamID int) ProjectState {
	return ProjectState{
		Name: p.Name,
		Team: teamID,
		Bgp: ProjectBgpState{
			Enabled:  p.Bgp.Enabled,
			LocalASN: p.Bgp.LocalASN,
		},
	}
}

func (*Project) WireDependencies(
	f infer.FieldSelector, args *ProjectArgs, state *ProjectState) {
	f.OutputField(&state.Name).DependsOn(f.InputField(&args.Name))
	f.OutputField(&state.Team).DependsOn(f.InputField(&args.Team))
	f.OutputField(&state.Bgp).DependsOn(f.InputField(&args.Bgp))
}

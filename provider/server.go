package provider

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"

	"github.com/cherryservers/cherrygo/v3"
	prov "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

type ServerClient interface {
	cherrygo.ServersService
}

type ServerClientFactory func(ctx context.Context) (ServerClient, error)

type Server struct {
	GetClient ServerClientFactory

	// DeploymentContext builds the context for server deployment polling.
	DeploymentContext func(context.Context) (context.Context, context.CancelFunc)

	DeploymentPollInterval DurationFunc

	GetLogger GetLoggerFunc
}

func (s *Server) Annotate(a infer.Annotator) {
	a.Describe(&s, "Cherry Servers server instance.")
}

type ServerArgs struct {
	Plan            string            `pulumi:"plan"`
	Project         int               `pulumi:"project"`
	Region          string            `pulumi:"region"`
	Hostname        string            `pulumi:"hostname,optional"`
	Image           string            `pulumi:"image,optional"`
	SSHKeys         []int             `pulumi:"sshKeys,optional"`
	ExtraIPs        []string          `pulumi:"extraIPs,optional"`
	UserData        string            `pulumi:"userData,optional"`
	Tags            map[string]string `pulumi:"tags,optional"`
	Spot            bool              `pulumi:"spot,optional"`
	OSPartitionSize int               `pulumi:"osPartitionSize,optional"`
	Cycle           string            `pulumi:"cycle,optional"`
	DiscountCode    string            `pulumi:"discountCode,optional"`
	Storage         int               `pulumi:"storage,optional"`
	BGP             bool              `pulumi:"bgp,optional"`
	AllowReinstall  bool              `pulumi:"allowReinstall,optional"`
}

func (s *ServerArgs) Annotate(a infer.Annotator) {
	a.Describe(&s.Plan, "Server plan slug.")
	a.Describe(&s.Project, "ID of the server the project belongs to.")
	a.Describe(&s.Region, "Server region slug.")
	a.Describe(&s.Hostname, "Server hostname.")
	a.Describe(&s.Image, "Server image slug. Updating requires re-installation.")
	a.Describe(&s.SSHKeys, "Server SSH key IDs. Updating requires re-installation.")
	a.Describe(&s.ExtraIPs, "IDs of extra IP addresses assigned to the server.")
	a.Describe(&s.UserData, "Server user data. Bash or cloud-config script. Updating requires re-installation.")
	a.Describe(&s.Tags, "Server tags.")
	a.Describe(&s.Spot, "Whether the server is a spot instance.")
	a.Describe(&s.OSPartitionSize, "Server OS partition size. Updating requires re-installation.")
	a.Describe(&s.Cycle, "Server billing cycle.")
	a.Describe(&s.DiscountCode, "Server discount code.")
	a.Describe(&s.Storage, "Server elastic block storage ID.")
	a.Describe(&s.BGP, "Whether BGP is enabled for the server.")
	a.Describe(&s.AllowReinstall, "Whether re-installation is permitted for this server.")
}

// ensureSorted ensures sortable arguments are sorted.
func (s *ServerArgs) ensureSorted() {
	slices.Sort(s.SSHKeys)
	slices.Sort(s.ExtraIPs)
}

// replacementInducing returns the input fields that cause resource replacement.
func (s *ServerArgs) replacementInducing(f infer.FieldSelector) []infer.InputField {
	return []infer.InputField{
		f.InputField(&s.Plan),
		f.InputField(&s.Project),
		f.InputField(&s.Region),
		f.InputField(&s.ExtraIPs),
		f.InputField(&s.Spot),
		f.InputField(&s.Cycle),
		f.InputField(&s.DiscountCode),
	}
}

type ServerPricingState struct {
	Price    float64 `pulumi:"price"`
	Currency string  `pulumi:"currency"`
	Unit     string  `pulumi:"unit"`
}

type ServerIPState struct {
	ID      string `pulumi:"id"`
	Type    string `pulumi:"type"`
	Address string `pulumi:"address"`
}

type ServerState struct {
	ServerArgs

	IPs     []ServerIPState    `pulumi:"ips"`
	Status  string             `pulumi:"status"`
	Pricing ServerPricingState `pulumi:"pricing"`
}

func (s *ServerState) Annotate(a infer.Annotator) {
	s.ServerArgs.Annotate(a)
	a.Describe(&s.IPs, "Server IP addresses.")
	a.Describe(&s.Status, "Server status, such as 'deploying' or 'deployed'.")
	a.Describe(&s.Pricing, "Server pricing.")
}

var (
	_ infer.Annotated                                     = (*Server)(nil)
	_ infer.Annotated                                     = (*ServerArgs)(nil)
	_ infer.Annotated                                     = (*ServerState)(nil)
	_ infer.CustomCreate[ServerArgs, ServerState]         = (*Server)(nil)
	_ infer.CustomDelete[ServerState]                     = (*Server)(nil)
	_ infer.CustomCheck[ServerArgs]                       = (*Server)(nil)
	_ infer.CustomUpdate[ServerArgs, ServerState]         = (*Server)(nil)
	_ infer.CustomDiff[ServerArgs, ServerState]           = (*Server)(nil)
	_ infer.CustomRead[ServerArgs, ServerState]           = (*Server)(nil)
	_ infer.ExplicitDependencies[ServerArgs, ServerState] = (*Server)(nil)
)

func (s *Server) Create(ctx context.Context, req infer.CreateRequest[ServerArgs]) (
	infer.CreateResponse[ServerState], error) {
	if req.DryRun {
		return infer.CreateResponse[ServerState]{
			Output: ServerState{
				ServerArgs: req.Inputs,
			},
		}, nil
	}

	client, err := s.GetClient(ctx)
	if err != nil {
		return infer.CreateResponse[ServerState]{}, err
	}

	sshKeyIDs := make([]string, len(req.Inputs.SSHKeys))
	for i, v := range req.Inputs.SSHKeys {
		sshKeyIDs[i] = strconv.Itoa(v)
	}

	server, _, err := client.Create(&cherrygo.CreateServer{
		ProjectID:       req.Inputs.Project,
		Plan:            req.Inputs.Plan,
		Hostname:        req.Inputs.Hostname,
		Image:           req.Inputs.Image,
		Region:          req.Inputs.Region,
		SSHKeys:         sshKeyIDs,
		IPAddresses:     req.Inputs.ExtraIPs,
		UserData:        base64.StdEncoding.EncodeToString([]byte(req.Inputs.UserData)),
		Tags:            &req.Inputs.Tags,
		SpotInstance:    req.Inputs.Spot,
		OSPartitionSize: req.Inputs.OSPartitionSize,
		StorageID:       req.Inputs.Storage,
		Cycle:           req.Inputs.Cycle,
	})
	if err != nil {
		return infer.CreateResponse[ServerState]{}, err
	}

	server, err = s.untilDeployed(ctx, server, client)
	if err != nil {
		return infer.CreateResponse[ServerState]{},
			fmt.Errorf("server %d didn't deploy: %w", server.ID, err)
	}

	// Unfortunately, the create request doesn't have a field for BGP,
	// so we may need another request, if BGP needs to be set.
	if req.Inputs.BGP != server.BGP.Enabled {
		server, _, err = client.Update(server.ID, &cherrygo.UpdateServer{
			Bgp: req.Inputs.BGP,
		})
		if err != nil {
			return infer.CreateResponse[ServerState]{},
				fmt.Errorf("failed to enable server %d BGP: %w", server.ID, err)
		}
	}

	return infer.CreateResponse[ServerState]{
		ID:     strconv.Itoa(server.ID),
		Output: serverStateFromClientResp(server, req.Inputs),
	}, nil
}

func (s *Server) Delete(ctx context.Context, req infer.DeleteRequest[ServerState]) (infer.DeleteResponse, error) {
	client, err := s.GetClient(ctx)
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	id, err := strconv.Atoi(req.ID)
	if err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("server id %q not parsable to int: %w", req.ID, err)
	}

	_, r, err := client.Delete(id)
	if err != nil && r.StatusCode == http.StatusNotFound {
		s.GetLogger(ctx).Warningf("server %s already deleted", req.ID)
		err = nil
	}
	return infer.DeleteResponse{}, err
}

func (s *Server) Check(ctx context.Context, req infer.CheckRequest) (
	infer.CheckResponse[ServerArgs], error) {
	args, failures, err := infer.DefaultCheck[ServerArgs](ctx, req.NewInputs)
	if err != nil {
		return infer.CheckResponse[ServerArgs]{
			Inputs:   args,
			Failures: failures,
		}, err
	}

	args.Hostname, err = autoname(args.Hostname, req.Name, req.OldInputs.Get("hostname"))
	return infer.CheckResponse[ServerArgs]{
		Inputs:   args,
		Failures: failures,
	}, err
}

// Update updates a server with a basic update request, a re-install, or both.
func (s *Server) Update(
	ctx context.Context, req infer.UpdateRequest[ServerArgs, ServerState]) (
	infer.UpdateResponse[ServerState], error) {
	if req.DryRun {
		return infer.UpdateResponse[ServerState]{
			Output: ServerState{
				ServerArgs: req.Inputs,
			},
		}, nil
	}

	req.Inputs.ensureSorted()

	client, err := s.GetClient(ctx)
	if err != nil {
		return infer.UpdateResponse[ServerState]{}, err
	}

	state, err := s.reinstall(ctx, req, client)
	if err != nil {
		return infer.UpdateResponse[ServerState]{},
			fmt.Errorf("failed to re-install server %q: %w", req.ID, err)
	}
	req.State = state

	state, err = updateServer(req, client)
	if err != nil {
		return infer.UpdateResponse[ServerState]{},
			fmt.Errorf("failed to re-update server %q: %w", req.ID, err)
	}

	return infer.UpdateResponse[ServerState]{
		Output: state,
	}, nil
}

// reinstall check if a reinstall is actually required and does nothing,
// if it isn't.
func (s *Server) reinstall(ctx context.Context, req infer.UpdateRequest[ServerArgs, ServerState], client ServerClient) (
	ServerState, error) {
	if !reinstallNeeded(req.Inputs, req.State.ServerArgs) {
		return req.State, nil
	}

	if !req.Inputs.AllowReinstall {
		return ServerState{}, errors.New(
			"`AllowReinstall` needs to be allowed to update " +
				"`Image`, `SSHKeys`, `UserData` or `OSPartitionSize`")
	}

	id, err := strconv.Atoi(req.ID)
	if err != nil {
		return ServerState{}, fmt.Errorf("server ID %q not parsable to int: %w", req.ID, err)
	}

	sshKeyIDs := make([]string, len(req.Inputs.SSHKeys))
	for i, v := range req.Inputs.SSHKeys {
		sshKeyIDs[i] = strconv.Itoa(v)
	}

	server, _, err := client.Reinstall(id, &cherrygo.ReinstallServerFields{
		Image:           req.Inputs.Image,
		SSHKeys:         sshKeyIDs,
		UserData:        req.Inputs.UserData,
		OSPartitionSize: req.Inputs.OSPartitionSize,
	})

	if err != nil {
		return ServerState{}, err
	}

	server, err = s.untilDeployed(ctx, server, client)
	if err != nil {
		return ServerState{}, fmt.Errorf("server %d didn't deploy: %w", server.ID, err)
	}

	return serverStateFromClientResp(server, req.Inputs), nil
}

func reinstallNeeded(inputs, state ServerArgs) bool {
	// Reinstall fields: Image, SSHKeys, UserData, OSPartitionSize.
	if inputs.Image != state.Image ||
		!slices.Equal(inputs.SSHKeys, state.SSHKeys) ||
		inputs.UserData != state.UserData ||
		inputs.OSPartitionSize != state.OSPartitionSize {
		return true
	}
	return false
}

// updateServer checks if an update is actually needed and does nothing, if it isn't.
func updateServer(req infer.UpdateRequest[ServerArgs, ServerState], client ServerClient) (
	ServerState, error) {
	if !serverUpdateNeeded(req.Inputs, req.State.ServerArgs) {
		return req.State, nil
	}

	id, err := strconv.Atoi(req.ID)
	if err != nil {
		return ServerState{}, fmt.Errorf("server ID %q not parsable to int: %w", req.ID, err)
	}

	server, _, err := client.Update(id, &cherrygo.UpdateServer{
		Hostname: req.Inputs.Hostname,
		Tags:     &req.Inputs.Tags,
		Bgp:      req.Inputs.BGP,
	})

	if err != nil {
		return ServerState{}, err
	}

	return serverStateFromClientResp(server, req.Inputs), nil
}

func serverUpdateNeeded(inputs, state ServerArgs) bool {
	if inputs.Hostname != state.Hostname ||
		!maps.Equal(inputs.Tags, state.Tags) ||
		inputs.BGP != state.BGP {
		return true
	}
	return false
}

func (s *Server) Diff(
	_ context.Context, req infer.DiffRequest[ServerArgs, ServerState]) (
	infer.DiffResponse, error) {
	diff := map[string]prov.PropertyDiff{}
	req.Inputs.ensureSorted()
	req.State.ServerArgs.ensureSorted()

	if req.Inputs.Plan != req.State.Plan {
		diff["plan"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.Project != req.State.Project {
		diff["project"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.Region != req.State.Region {
		diff["region"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.Hostname != req.State.Hostname {
		diff["hostname"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if req.Inputs.Image != req.State.Image {
		diff["image"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if !slices.Equal(req.Inputs.SSHKeys, req.State.SSHKeys) {
		diff["sshKeys"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if !slices.Equal(req.Inputs.ExtraIPs, req.State.ExtraIPs) {
		diff["extraIPs"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.UserData != req.State.UserData {
		diff["userData"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if maps.Equal(req.Inputs.Tags, req.State.Tags) {
		diff["tags"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if req.Inputs.Spot != req.State.Spot {
		diff["spot"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.OSPartitionSize != req.State.OSPartitionSize {
		diff["osPartitionSize"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if req.Inputs.Cycle != req.State.Cycle {
		diff["cycle"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.DiscountCode != req.State.DiscountCode {
		diff["discountCode"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.Storage != req.State.Storage {
		diff["storage"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.BGP != req.State.BGP {
		diff["bgp"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if req.Inputs.AllowReinstall != req.State.AllowReinstall {
		diff["allowReinstall"] = prov.PropertyDiff{Kind: prov.Update}
	}

	return infer.DiffResponse{
		DeleteBeforeReplace: true,
		HasChanges:          len(diff) > 0,
		DetailedDiff:        diff,
	}, nil
}

func (s *Server) Read(
	ctx context.Context, req infer.ReadRequest[ServerArgs, ServerState]) (
	infer.ReadResponse[ServerArgs, ServerState], error) {
	client, err := s.GetClient(ctx)
	if err != nil {
		return infer.ReadResponse[ServerArgs, ServerState]{}, err
	}

	id, err := strconv.Atoi(req.ID)
	if err != nil {
		return infer.ReadResponse[ServerArgs, ServerState]{},
			fmt.Errorf("Server ID %q not parsable to int: %w", req.ID, err)
	}

	server, r, err := client.Get(id, nil)
	if err != nil && r.StatusCode == http.StatusNotFound {
		return infer.ReadResponse[ServerArgs, ServerState]{}, nil
	}

	return infer.ReadResponse[ServerArgs, ServerState]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  serverStateFromClientResp(server, req.Inputs),
	}, err
}

func serverStateFromClientResp(s cherrygo.Server, inputs ServerArgs) ServerState {
	inputs.ensureSorted()

	sshKeyIDs := make([]int, len(s.SSHKeys))
	for i, v := range s.SSHKeys {
		sshKeyIDs[i] = v.ID
	}

	ips := make([]ServerIPState, len(s.IPAddresses))
	for i, v := range s.IPAddresses {
		ips[i] = ServerIPState{
			ID:      v.ID,
			Type:    v.Type,
			Address: v.Address,
		}
	}

	return ServerState{
		ServerArgs: ServerArgs{
			Plan:            s.Plan.Slug,
			Project:         s.Project.ID,
			Region:          s.Region.Slug,
			Hostname:        s.Hostname,
			Image:           s.DeployedImage.Slug,
			SSHKeys:         sshKeyIDs,
			ExtraIPs:        inputs.ExtraIPs,
			UserData:        inputs.UserData,
			Tags:            s.Tags,
			Spot:            s.SpotInstance,
			OSPartitionSize: inputs.OSPartitionSize,
			Cycle:           inputs.Cycle,
			DiscountCode:    inputs.DiscountCode,
			Storage:         s.Storage.ID,
			BGP:             s.BGP.Enabled,
			AllowReinstall:  inputs.AllowReinstall,
		},
		IPs:    ips,
		Status: s.State,
		Pricing: ServerPricingState{
			Price:    float64(s.Pricing.Price),
			Currency: s.Pricing.Currency,
			Unit:     s.Pricing.Unit,
		},
	}
}

func (s *Server) WireDependencies(
	f infer.FieldSelector, args *ServerArgs, state *ServerState) {
	f.OutputField(&state.IPs).DependsOn(
		append(
			args.replacementInducing(f),
			f.InputField(&args.ExtraIPs))...)
	f.OutputField(&state.Pricing).DependsOn(
		f.InputField(&args.Plan),
		f.InputField(&args.Region),
		f.InputField(&args.DiscountCode))
	f.OutputField(&state.Plan).DependsOn(f.InputField(&args.Plan))
	f.OutputField(&state.Project).DependsOn(f.InputField(&args.Project))
	f.OutputField(&state.Region).DependsOn(f.InputField(&args.Region))
	f.OutputField(&state.Hostname).DependsOn(f.InputField(&args.Hostname))
	f.OutputField(&state.Image).DependsOn(f.InputField(&args.Image))
	f.OutputField(&state.SSHKeys).DependsOn(f.InputField(&args.SSHKeys))
	f.OutputField(&state.ExtraIPs).DependsOn(f.InputField(&args.ExtraIPs))
	f.OutputField(&state.UserData).DependsOn(f.InputField(&args.UserData))
	f.OutputField(&state.Tags).DependsOn(f.InputField(&args.Tags))
	f.OutputField(&state.Spot).DependsOn(f.InputField(&args.Spot))
	f.OutputField(&state.OSPartitionSize).DependsOn(f.InputField(&args.OSPartitionSize))
	f.OutputField(&state.Cycle).DependsOn(f.InputField(&args.Cycle))
	f.OutputField(&state.DiscountCode).DependsOn(f.InputField(&args.DiscountCode))
	f.OutputField(&state.Storage).DependsOn(f.InputField(&args.Storage))
	f.OutputField(&state.BGP).DependsOn(f.InputField(&args.BGP))
	f.OutputField(&state.AllowReinstall).DependsOn(f.InputField(&args.AllowReinstall))
}

func (s *Server) untilDeployed(
	ctx context.Context,
	server cherrygo.Server,
	client ServerClient) (cherrygo.Server, error) {
	ctx, cancel := s.DeploymentContext(ctx)
	defer cancel()

	var err error
	err = Until(ctx, NewTickSource(s.DeploymentPollInterval), func() (bool, error) {
		server, _, err = client.Get(server.ID, nil)
		return server.Status == "deployed", err
	})
	return server, err
}

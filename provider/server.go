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
	"strings"
	"time"

	"github.com/cherryservers/cherrygo/v3"
	prov "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/property"
)

type ServerClient interface {
	Create(*cherrygo.CreateServer) (cherrygo.Server, *cherrygo.Response, error)
	Update(int, *cherrygo.UpdateServer) (cherrygo.Server, *cherrygo.Response, error)
	Get(int, *cherrygo.GetOptions) (cherrygo.Server, *cherrygo.Response, error)
	Delete(int) (cherrygo.Server, *cherrygo.Response, error)
	Reinstall(int, *cherrygo.ReinstallServerFields) (cherrygo.Server, *cherrygo.Response, error)
}

type ServerClientFactory func(ctx context.Context) (ServerClient, error)

type ImageClient interface {
	// Get retrieves image slugs based on a plan slug.
	Get(string) ([]string, error)
}

type ImageClientFactory func(ctx context.Context) (ImageClient, error)

type Server struct {
	GetClient ServerClientFactory

	// DeploymentContext builds the context for server deployment polling.
	DeploymentContext func(context.Context) (context.Context, context.CancelFunc)

	GetLogger      GetLoggerFunc
	GetImageClient ImageClientFactory

	pollTicker func() <-chan time.Time
}

func (s *Server) Annotate(a infer.Annotator) {
	a.Describe(&s, "Cherry Servers server instance.")
}

type ServerArgs struct {
	Plan           string            `pulumi:"plan"`
	ProjectID      int               `pulumi:"projectID"`
	Region         string            `pulumi:"region"`
	Hostname       string            `pulumi:"hostname,optional"`
	Image          string            `pulumi:"image,optional"`
	SSHKeyIDs      []int             `pulumi:"sshKeyIDs,optional"`
	ExtraIPIDs     []string          `pulumi:"extraIPIDs,optional"`
	UserData       string            `pulumi:"userData,optional"`
	Tags           map[string]string `pulumi:"tags,optional"`
	Spot           bool              `pulumi:"spot,optional"`
	Cycle          string            `pulumi:"cycle,optional"`
	DiscountCode   string            `pulumi:"discountCode,optional"`
	BlockStorageID int               `pulumi:"blockStorageID,optional"`
	BGP            bool              `pulumi:"bgp,optional"`
	AllowReinstall bool              `pulumi:"allowReinstall,optional"`
}

func (s *ServerArgs) Annotate(a infer.Annotator) {
	a.Describe(&s.Plan, "Server plan slug.")
	a.Describe(&s.ProjectID, "ID of the server the project belongs to.")
	a.Describe(&s.Region, "Server region slug.")
	a.Describe(&s.Hostname, "Server hostname.")
	a.Describe(&s.Image, "Server image slug. Updating requires re-installation.")
	a.Describe(&s.SSHKeyIDs, "Server SSH key IDs. Updating requires re-installation.")
	a.Describe(&s.ExtraIPIDs, "IDs of extra IP addresses assigned to the server.")
	a.Describe(&s.UserData, "Server user data. Bash or cloud-config script. Updating requires re-installation.")
	a.Describe(&s.Tags, "Server tags.")
	a.Describe(&s.Spot, "Whether the server is a spot instance.")
	a.Describe(&s.Cycle, "Server billing cycle.")
	a.Describe(&s.DiscountCode, "Server discount code.")
	a.Describe(&s.BlockStorageID, "Server elastic block storage ID.")
	a.Describe(&s.BGP, "Whether BGP is enabled for the server.")
	a.Describe(&s.AllowReinstall, "Whether re-installation is permitted for this server.")
}

func (s *ServerArgs) canonicalize() {
	slices.Sort(s.SSHKeyIDs)
	slices.Sort(s.ExtraIPIDs)

	// Cherry Servers API silently converts hostnames to lowercase, so convert here, to
	// avoid state drift.
	s.Hostname = strings.ToLower(s.Hostname)
}

// replacementInducing returns the input fields that cause resource replacement.
func (s *ServerArgs) replacementInducing(f infer.FieldSelector) []infer.InputField {
	return []infer.InputField{
		f.InputField(&s.Plan),
		f.InputField(&s.ProjectID),
		f.InputField(&s.Region),
		f.InputField(&s.ExtraIPIDs),
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
	Pricing ServerPricingState `pulumi:"pricing"`
}

func (s *ServerState) canonicalize() {
	slices.Sort(s.SSHKeyIDs)
	slices.Sort(s.ExtraIPIDs)
	slices.SortFunc(s.IPs, func(a ServerIPState, b ServerIPState) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
}

func (s *ServerState) Annotate(a infer.Annotator) {
	s.ServerArgs.Annotate(a)
	a.Describe(&s.IPs, "Server IP addresses.")
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

	sshKeyIDs := make([]string, len(req.Inputs.SSHKeyIDs))
	for i, v := range req.Inputs.SSHKeyIDs {
		sshKeyIDs[i] = strconv.Itoa(v)
	}

	creationReq := cherrygo.CreateServer{
		ProjectID:    req.Inputs.ProjectID,
		Plan:         req.Inputs.Plan,
		Hostname:     req.Inputs.Hostname,
		Region:       req.Inputs.Region,
		SSHKeys:      sshKeyIDs,
		IPAddresses:  req.Inputs.ExtraIPIDs,
		UserData:     base64.StdEncoding.EncodeToString([]byte(req.Inputs.UserData)),
		Tags:         &req.Inputs.Tags,
		SpotInstance: req.Inputs.Spot,
		Cycle:        req.Inputs.Cycle,
		DiscountCode: req.Inputs.DiscountCode,
		Image:        req.Inputs.Image,
	}

	server, _, err := client.Create(&creationReq)
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

		// Need to do an extra request, because not all required fields
		// are returned on update.
		server, _, err = client.Get(server.ID, nil)
		if err != nil {
			return infer.CreateResponse[ServerState]{},
				fmt.Errorf("failed to get server %d: %w", server.ID, err)
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

	// Default to hourly cycle.
	if args.Cycle == "" {
		args.Cycle = "hourly"
	}

	if args.Tags == nil {
		args.Tags = make(map[string]string)
	}

	if args.ExtraIPIDs == nil {
		args.ExtraIPIDs = make([]string, 0)
	}

	if args.SSHKeyIDs == nil {
		args.SSHKeyIDs = make([]int, 0)
	}

	args.Image, err = s.checkImage(ctx, args.Image, args.Plan, req.OldInputs.Get("image"))
	if err != nil {
		return infer.CheckResponse[ServerArgs]{
			Inputs:   args,
			Failures: failures,
		}, err
	}

	args.Hostname, err = autoname(args.Hostname, req.Name, req.OldInputs.Get("hostname"))

	args.canonicalize()

	return infer.CheckResponse[ServerArgs]{
		Inputs:   args,
		Failures: failures,
	}, err
}

// Update updates a server with a basic update request, a re-install, or both.
func (s *Server) Update(
	ctx context.Context, req infer.UpdateRequest[ServerArgs, ServerState]) (
	infer.UpdateResponse[ServerState], error) {
	var ips []ServerIPState
	if !reinstallNeeded(req.Inputs, req.State.ServerArgs) {
		ips = req.State.IPs
	}

	if req.DryRun {
		return infer.UpdateResponse[ServerState]{
			Output: ServerState{
				// Pricing can only change on replacement.
				Pricing:    req.State.Pricing,
				ServerArgs: req.Inputs,
				IPs:        ips,
			},
		}, nil
	}

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

	sshKeyIDs := make([]string, len(req.Inputs.SSHKeyIDs))
	for i, v := range req.Inputs.SSHKeyIDs {
		sshKeyIDs[i] = strconv.Itoa(v)
	}

	server, _, err := client.Reinstall(id, &cherrygo.ReinstallServerFields{
		Image:    req.Inputs.Image,
		SSHKeys:  sshKeyIDs,
		UserData: req.Inputs.UserData,
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
		!slices.Equal(inputs.SSHKeyIDs, state.SSHKeyIDs) ||
		inputs.UserData != state.UserData {
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

	if req.Inputs.Plan != req.State.Plan {
		diff["plan"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.ProjectID != req.State.ProjectID {
		diff["projectID"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
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

	if !slices.Equal(req.Inputs.SSHKeyIDs, req.State.SSHKeyIDs) {
		diff["sshKeyIDs"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if !slices.Equal(req.Inputs.ExtraIPIDs, req.State.ExtraIPIDs) {
		diff["extraIPIDs"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.UserData != req.State.UserData {
		diff["userData"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if !maps.Equal(req.Inputs.Tags, req.State.Tags) {
		diff["tags"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if req.Inputs.Spot != req.State.Spot {
		diff["spot"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.Cycle != req.State.Cycle {
		diff["cycle"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.DiscountCode != req.State.DiscountCode {
		diff["discountCode"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.BlockStorageID != req.State.BlockStorageID {
		diff["blockStorageID"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
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

	args := ServerArgs{
		Plan:       s.Plan.Slug,
		ProjectID:  s.Project.ID,
		Region:     s.Region.Slug,
		Hostname:   s.Hostname,
		SSHKeyIDs:  sshKeyIDs,
		ExtraIPIDs: inputs.ExtraIPIDs,
		UserData:   inputs.UserData,
		Tags:       s.Tags,
		Spot:       s.SpotInstance,
		// Don't use 'cycle' from the response, because it doesn't
		// use the same form (it's capitalized).
		Cycle:          inputs.Cycle,
		DiscountCode:   inputs.DiscountCode,
		BGP:            s.BGP.Enabled,
		AllowReinstall: inputs.AllowReinstall,
		BlockStorageID: s.Storage.ID,
		Image:          s.DeployedImage.Slug,
	}

	state := ServerState{
		ServerArgs: args,
		IPs:        ips,
		Pricing: ServerPricingState{
			Price:    float64(s.Pricing.UnitPrice),
			Currency: s.Pricing.Currency,
			Unit:     s.Pricing.Unit,
		},
	}

	state.canonicalize()

	return state
}

func (s *Server) WireDependencies(
	f infer.FieldSelector, args *ServerArgs, state *ServerState) {
	f.OutputField(&state.IPs).DependsOn(args.replacementInducing(f)...)
	f.OutputField(&state.Pricing).DependsOn(
		f.InputField(&args.Plan),
		f.InputField(&args.Region),
		f.InputField(&args.DiscountCode),
		f.InputField(&args.Cycle),
	)
	f.OutputField(&state.Plan).DependsOn(f.InputField(&args.Plan))
	f.OutputField(&state.ProjectID).DependsOn(f.InputField(&args.ProjectID))
	f.OutputField(&state.Region).DependsOn(f.InputField(&args.Region))
	f.OutputField(&state.Hostname).DependsOn(f.InputField(&args.Hostname))
	f.OutputField(&state.Image).DependsOn(f.InputField(&args.Image))
	f.OutputField(&state.SSHKeyIDs).DependsOn(f.InputField(&args.SSHKeyIDs))
	f.OutputField(&state.ExtraIPIDs).DependsOn(f.InputField(&args.ExtraIPIDs))
	f.OutputField(&state.UserData).DependsOn(f.InputField(&args.UserData))
	f.OutputField(&state.Tags).DependsOn(f.InputField(&args.Tags))
	f.OutputField(&state.Spot).DependsOn(f.InputField(&args.Spot))
	f.OutputField(&state.Cycle).DependsOn(f.InputField(&args.Cycle))
	f.OutputField(&state.DiscountCode).DependsOn(f.InputField(&args.DiscountCode))
	f.OutputField(&state.BlockStorageID).DependsOn(f.InputField(&args.BlockStorageID))
	f.OutputField(&state.BGP).DependsOn(f.InputField(&args.BGP))
	f.OutputField(&state.AllowReinstall).DependsOn(f.InputField(&args.AllowReinstall))
}

func (s *Server) untilDeployed(
	ctx context.Context,
	server cherrygo.Server,
	client ServerClient) (cherrygo.Server, error) {
	ctx, cancel := s.DeploymentContext(ctx)
	defer cancel()

	ticker := s.pollTicker()

	for {
		select {
		case <-ctx.Done():
			return cherrygo.Server{}, ctx.Err()
		case <-ticker:
			server, _, err := client.Get(server.ID, nil)
			if err != nil {
				return cherrygo.Server{}, err
			}
			if server.Status == "deployed" {
				return server, nil
			}
		}
	}
}

func (s *Server) defaultImage(ctx context.Context, plan string) (string, error) {
	// Prefer Ubuntu.
	const defaultPrefix = "ubuntu"

	client, err := s.GetImageClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get image client: %w", err)
	}

	images, err := client.Get(plan)
	if err != nil {
		return "", fmt.Errorf("failed to get plan %q images: %w", plan, err)
	}

	if len(images) == 0 {
		return "", fmt.Errorf("no images for plan %q", plan)
	}

	image := images[0]

	// Try to get the latest distribution.
	for _, img := range images {
		if strings.HasPrefix(img, defaultPrefix) {
			if img > image || !strings.HasPrefix(image, defaultPrefix) {
				image = img
			}
		}
	}

	return image, nil
}

func (s *Server) checkImage(ctx context.Context, arg, plan string, old property.Value) (string, error) {
	if arg != "" {
		return arg, nil
	}

	if old.IsString() && old.AsString() != "" {
		return old.AsString(), nil
	}

	return s.defaultImage(ctx, plan)
}

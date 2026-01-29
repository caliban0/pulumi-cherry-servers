package provider

import (
	"context"
	"maps"
	"net/http"
	"strconv"

	"github.com/cherryservers/cherrygo/v3"
	prov "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

type IPClient interface {
	cherrygo.IpAddressesService
}

type IPClientFactory func(cfg Config) (IPClient, error)

type IP struct {
	GetClient IPClientFactory
}

func (i *IP) Annotate(a infer.Annotator) {
	a.Describe(&i, "Cherry Servers IP address.")
}

type IPArgs struct {
	Region     string            `pulumi:"region"`
	Project    int               `pulumi:"project"`
	PTRRecord  string            `pulumi:"ptrRecord,optional"`
	ARecord    string            `pulumi:"aRecord,optional"`
	RoutedTo   string            `pulumi:"routedTo,optional"`
	TargetedTo int               `pulumi:"targetedTo,optional"`
	Tags       map[string]string `pulumi:"tags,optional"`
}

func (i *IPArgs) Annotate(a infer.Annotator) {
	a.Describe(&i.Region, "IP address region slug.")
	a.Describe(&i.Region, "IP address project ID.")
	a.Describe(&i.PTRRecord, "Reverse DNS name for the IP address."+
		"The value will be stored in `ptrRecordEffective`")
	a.Describe(&i.ARecord, "Relative DNS name for the IP address."+
		"Resulting FQDN will be '<relative-dns-name>.cloud.cherryservers.net'"+
		" and must be globally unique. The value will be stored in `aRecordEffective`")
	a.Describe(&i.RoutedTo, "IP address that this address is routed to.")
	a.Describe(&i.TargetedTo, "Server that this address is targeted to.")
	a.Describe(&i.Tags, "IP address tags.")
}

type IPState struct {
	IPArgs
	Address            string `pulumi:"address"`
	AddressFamily      int    `pulumi:"addressFamily"`
	CIDR               string `pulumi:"cidr"`
	Type               string `pulumi:"type"`
	ARecordEffective   string `pulumi:"aRecordEffective"`
	PTRRecordEffective string `pulumi:"ptrRecordEffective"`
}

func (i *IPState) Annotate(a infer.Annotator) {
	i.IPArgs.Annotate(a)
	a.Describe(&i.Address, "Actual address.")
	a.Describe(&i.AddressFamily, "IP address family.")
	a.Describe(&i.CIDR, "IP address CIDR.")
	a.Describe(&i.Type, "IP address type.")
	a.Describe(&i.ARecordEffective, "Relative DNS name for the IP address.")
	a.Describe(&i.PTRRecordEffective, "Reverse DNS name for the IP address.")
}

var (
	_ infer.Annotated                             = (*IP)(nil)
	_ infer.Annotated                             = (*IPArgs)(nil)
	_ infer.Annotated                             = (*IPState)(nil)
	_ infer.CustomCreate[IPArgs, IPState]         = (*IP)(nil)
	_ infer.CustomDelete[IPState]                 = (*IP)(nil)
	_ infer.CustomCheck[IPArgs]                   = (*IP)(nil)
	_ infer.CustomUpdate[IPArgs, IPState]         = (*IP)(nil)
	_ infer.CustomDiff[IPArgs, IPState]           = (*IP)(nil)
	_ infer.CustomRead[IPArgs, IPState]           = (*IP)(nil)
	_ infer.ExplicitDependencies[IPArgs, IPState] = (*IP)(nil)
)

func (i *IP) Create(ctx context.Context, req infer.CreateRequest[IPArgs]) (
	infer.CreateResponse[IPState], error) {
	if req.DryRun {
		return infer.CreateResponse[IPState]{
			Output: IPState{
				IPArgs: req.Inputs,
			},
		}, nil
	}

	client, err := i.GetClient(infer.GetConfig[Config](ctx))
	if err != nil {
		return infer.CreateResponse[IPState]{}, err
	}

	ip, _, err := client.Create(req.Inputs.Project, &cherrygo.CreateIPAddress{
		Region:     req.Inputs.Region,
		PtrRecord:  req.Inputs.PTRRecord,
		ARecord:    req.Inputs.ARecord,
		RoutedTo:   req.Inputs.RoutedTo,
		TargetedTo: strconv.Itoa(req.Inputs.TargetedTo),
		Tags:       &req.Inputs.Tags,
	})
	if err != nil {
		return infer.CreateResponse[IPState]{}, err
	}

	return infer.CreateResponse[IPState]{
		ID:     ip.ID,
		Output: ipStateFromClientResp(ip, req.Inputs.Project, req.Inputs.ARecord, req.Inputs.PTRRecord),
	}, nil
}

func (i *IP) Delete(ctx context.Context, req infer.DeleteRequest[IPState]) (infer.DeleteResponse, error) {
	client, err := i.GetClient(infer.GetConfig[Config](ctx))
	if err != nil {
		return infer.DeleteResponse{}, err
	}

	r, err := client.Remove(req.ID)
	if err != nil && r.StatusCode == http.StatusNotFound {
		prov.GetLogger(ctx).Warningf("ip address %s already deleted", req.ID)
		err = nil
	}
	return infer.DeleteResponse{}, err
}

func (i *IP) Check(ctx context.Context, req infer.CheckRequest) (infer.CheckResponse[IPArgs], error) {
	args, failures, err := infer.DefaultCheck[IPArgs](ctx, req.NewInputs)
	if err != nil {
		return infer.CheckResponse[IPArgs]{
			Inputs:   args,
			Failures: failures,
		}, err
	}

	// RoutedTo and TargetedTo are mutually exclusive.
	if args.RoutedTo != "" && args.TargetedTo != 0 {
		failures = append(failures,
			prov.CheckFailure{
				Property: "routedTo", Reason: "'routedTo' is mutually exclusive with 'targetedTo'"},
			prov.CheckFailure{
				Property: "targetedTo", Reason: "'targetedTo' is mutually exclusive with 'routedTo'"})
	}

	return infer.CheckResponse[IPArgs]{
		Inputs:   args,
		Failures: failures,
	}, err
}

func (i *IP) Update(
	ctx context.Context, req infer.UpdateRequest[IPArgs, IPState]) (
	infer.UpdateResponse[IPState], error) {
	if req.DryRun {
		return infer.UpdateResponse[IPState]{
			Output: IPState{
				IPArgs: req.Inputs,
			},
		}, nil
	}

	client, err := i.GetClient(infer.GetConfig[Config](ctx))
	if err != nil {
		return infer.UpdateResponse[IPState]{}, err
	}

	ip, _, err := client.Update(req.ID, &cherrygo.UpdateIPAddress{
		PtrRecord:  req.Inputs.PTRRecord,
		ARecord:    req.Inputs.ARecord,
		RoutedTo:   req.Inputs.RoutedTo,
		TargetedTo: strconv.Itoa(req.Inputs.TargetedTo),
		Tags:       &req.Inputs.Tags,
	})

	return infer.UpdateResponse[IPState]{
		Output: ipStateFromClientResp(ip, req.Inputs.Project, req.Inputs.ARecord, req.Inputs.PTRRecord),
	}, err
}

func (i *IP) Diff(
	_ context.Context, req infer.DiffRequest[IPArgs, IPState]) (
	infer.DiffResponse, error) {
	diff := map[string]prov.PropertyDiff{}

	if req.Inputs.Region != req.State.Region {
		diff["region"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.Project != req.State.Project {
		diff["project"] = prov.PropertyDiff{Kind: prov.UpdateReplace}
	}

	if req.Inputs.PTRRecord != req.State.PTRRecord {
		diff["ptrRecord"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if req.Inputs.ARecord != req.State.ARecord {
		diff["aRecord"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if req.Inputs.RoutedTo != req.State.RoutedTo {
		diff["routedTo"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if req.Inputs.TargetedTo != req.State.TargetedTo {
		diff["targetedTo"] = prov.PropertyDiff{Kind: prov.Update}
	}

	if !maps.Equal(req.Inputs.Tags, req.State.Tags) {
		diff["tags"] = prov.PropertyDiff{Kind: prov.Update}
	}

	return infer.DiffResponse{
		DeleteBeforeReplace: true,
		HasChanges:          len(diff) > 0,
		DetailedDiff:        diff,
	}, nil
}

func (i *IP) Read(
	ctx context.Context, req infer.ReadRequest[IPArgs, IPState]) (
	infer.ReadResponse[IPArgs, IPState], error) {
	client, err := i.GetClient(infer.GetConfig[Config](ctx))
	if err != nil {
		return infer.ReadResponse[IPArgs, IPState]{}, err
	}

	ip, r, err := client.Get(req.ID, nil)
	if err != nil && r.StatusCode == http.StatusNotFound {
		return infer.ReadResponse[IPArgs, IPState]{}, nil
	}

	return infer.ReadResponse[IPArgs, IPState]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  ipStateFromClientResp(ip, req.Inputs.Project, req.Inputs.ARecord, req.Inputs.ARecord),
	}, err
}

func ipStateFromClientResp(ip cherrygo.IPAddress, projectID int, aRecordInput, ptrRecordInput string) IPState {
	tags := &map[string]string{}
	if ip.Tags != nil {
		tags = ip.Tags
	}

	return IPState{
		IPArgs: IPArgs{
			Region:     ip.Region.Slug,
			Project:    projectID,
			PTRRecord:  ptrRecordInput,
			ARecord:    aRecordInput,
			RoutedTo:   ip.RoutedTo.ID,
			TargetedTo: ip.TargetedTo.ID,
			Tags:       *tags,
		},
		Address:            ip.Address,
		AddressFamily:      ip.AddressFamily,
		CIDR:               ip.Cidr,
		Type:               ip.Type,
		ARecordEffective:   ip.ARecord,
		PTRRecordEffective: ip.PtrRecord,
	}
}

func (*IP) WireDependencies(
	f infer.FieldSelector, args *IPArgs, state *IPState) {
	f.OutputField(&state.Region).DependsOn(f.InputField(&args.Region))
	f.OutputField(&state.Project).DependsOn(f.InputField(&args.Project))
	f.OutputField(&state.PTRRecord).DependsOn(f.InputField(&args.PTRRecord))
	f.OutputField(&state.ARecord).DependsOn(f.InputField(&args.ARecord))
	f.OutputField(&state.RoutedTo).DependsOn(f.InputField(&args.RoutedTo), f.InputField(&args.TargetedTo))
	f.OutputField(&state.TargetedTo).DependsOn(f.InputField(&args.RoutedTo), f.InputField(&args.TargetedTo))
	f.OutputField(&state.Tags).DependsOn(f.InputField(&args.Tags))
	f.OutputField(&state.Address).DependsOn(f.InputField(&args.Region), f.InputField(&args.Project))
	f.OutputField(&state.AddressFamily).DependsOn(f.InputField(&args.Region), f.InputField(&args.Project))
	f.OutputField(&state.CIDR).DependsOn(f.InputField(&args.Region), f.InputField(&args.Project))
	f.OutputField(&state.ARecordEffective).DependsOn(f.InputField(&args.ARecord))
	f.OutputField(&state.PTRRecordEffective).DependsOn(f.InputField(&args.PTRRecord))
}

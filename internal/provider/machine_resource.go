package provider

import (
	"context"
	"fmt"
	"errors"
	"github.com/fly-apps/terraform-provider-fly/pkg/apiv1"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"net/http"
)

var _ tfsdk.ResourceType = flyMachineResourceType{}
var _ tfsdk.Resource = flyMachineResource{}
var _ tfsdk.ResourceWithImportState = flyMachineResource{}

type flyMachineResourceType struct{}

type flyMachineResource struct {
	provider provider
	endpoint string
}

type TfPort struct {
	Port     types.Int64    `tfsdk:"port"`
	Handlers []types.String `tfsdk:"handlers"`
}

type TfService struct {
	Ports        []TfPort     `tfsdk:"ports"`
	Protocol     types.String `tfsdk:"protocol"`
	InternalPort types.Int64  `tfsdk:"internal_port"`
}

type flyMachineResourceData struct {
	Name       types.String `tfsdk:"name"`
	Region     types.String `tfsdk:"region"`
	Id         types.String `tfsdk:"id"`
	App        types.String `tfsdk:"app"`
	Image      types.String `tfsdk:"image"`
	Cpus       types.Int64  `tfsdk:"cpus"`
	MemoryMb   types.Int64  `tfsdk:"memorymb"`
	CpuType    types.String `tfsdk:"cputype"`
	Env        types.Map    `tfsdk:"env"`
	Cmd        []string     `tfsdk:"cmd"`
	Entrypoint []string     `tfsdk:"entrypoint"`

	Mounts   []TfMachineMount `tfsdk:"mounts"`
	Services []TfService      `tfsdk:"services"`
}

type TfMachineMount struct {
	Encrypted types.Bool   `tfsdk:"encrypted"`
	Path      types.String `tfsdk:"path"`
	SizeGb    types.Int64  `tfsdk:"size_gb"`
	Volume    types.String `tfsdk:"volume"`
}

func KVToTfMap(kv map[string]string, elemType attr.Type) types.Map {
	var TFMap types.Map
	TFMap.ElemType = elemType
	for key, value := range kv {
		if TFMap.Elems == nil {
			TFMap.Elems = map[string]attr.Value{}
		}
		TFMap.Elems[key] = types.String{Value: value}
	}
	return TFMap
}

func (mr flyMachineResourceType) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Fly machine resource",
		Attributes: map[string]tfsdk.Attribute{
			"name": {
				MarkdownDescription: "machine name",
				Optional:            true,
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				Type: types.StringType,
			},
			"region": {
				MarkdownDescription: "machine region",
				Required:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					tfsdk.RequiresReplace(),
				},
				Type: types.StringType,
			},
			"id": {
				MarkdownDescription: "machine id",
				Computed:            true,
				Type:                types.StringType,
			},
			"app": {
				MarkdownDescription: "fly app",
				Optional:            true,
				Type:                types.StringType,
			},
			"cmd": {
				MarkdownDescription: "exec command",
				Optional:            true,
				//Computed:            true,
				Type: types.ListType{ElemType: types.StringType},
			},
			"entrypoint": {
				MarkdownDescription: "image entrypoint",
				Optional:            true,
				//Computed:            true,
				Type: types.ListType{ElemType: types.StringType},
			},
			"image": {
				MarkdownDescription: "docker image",
				Required:            true,
				Type:                types.StringType,
			},
			"cputype": {
				MarkdownDescription: "cpu type",
				Computed:            true,
				Optional:            true,
				Type:                types.StringType,
			},
			"cpus": {
				MarkdownDescription: "cpu count",
				Computed:            true,
				Optional:            true,
				Type:                types.Int64Type,
			},
			"memorymb": {
				MarkdownDescription: "memory mb",
				Computed:            true,
				Optional:            true,
				Type:                types.Int64Type,
			},
			"env": {
				MarkdownDescription: "Optional environment variables, keys and values must be strings",
				Optional:            true,
				Computed:            true,
				Type:                types.MapType{ElemType: types.StringType},
			},
			"mounts": {
				MarkdownDescription: "Volume mounts",
				Optional:            true,
				Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
					"encrypted": {
						Optional: true,
						Computed: true,
						Type:     types.BoolType,
					},
					"path": {
						Required:            true,
						MarkdownDescription: "Path for volume to be mounted on vm",
						Type:                types.StringType,
					},
					"size_gb": {
						Optional: true,
						Computed: true,
						Type:     types.Int64Type,
					},
					"volume": {
						Required:            true,
						MarkdownDescription: "Name or ID of volume",
						Type:                types.StringType,
					},
				}),
			},
			"services": {
				MarkdownDescription: "services",
				Optional:            true,
				Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
					"ports": {
						MarkdownDescription: "External ports and handlers",
						Required:            true,
						Attributes: tfsdk.ListNestedAttributes(map[string]tfsdk.Attribute{
							"port": {
								MarkdownDescription: "External port",
								Required:            true,
								Type:                types.Int64Type,
							},
							"handlers": {
								MarkdownDescription: "How the edge should process requests",
								Required:            true,
								Type:                types.ListType{ElemType: types.StringType},
							},
						}),
					},
					"protocol": {
						MarkdownDescription: "network protocol",
						Required:            true,
						Type:                types.StringType,
					},
					"internal_port": {
						MarkdownDescription: "Port application listens on internally",
						Required:            true,
						Type:                types.Int64Type,
					},
				}),
			},
		},
	}, nil
}

func (mr flyMachineResourceType) NewResource(_ context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return flyMachineResource{
		provider: provider,
	}, diags
}

func (mr flyMachineResource) ValidateOpenTunnel() (bool, error) {
	_, err := mr.provider.httpClient.R().Get(fmt.Sprintf("http://%s", mr.provider.httpEndpoint))
	if err != nil {
		return false, errors.New("Can't connect to the api, is the tunnel open? :)")
	}
	return true, nil
}

func TfServicesToServices(input []TfService) []apiv1.Service {
	services := make([]apiv1.Service, 0)
	for _, s := range input {
		var ports []apiv1.Port
		for _, j := range s.Ports {
			var handlers []string
			for _, k := range j.Handlers {
				handlers = append(handlers, k.Value)
			}
			ports = append(ports, apiv1.Port{
				Port:     j.Port.Value,
				Handlers: handlers,
			})
		}
		services = append(services, apiv1.Service{
			Ports:        ports,
			Protocol:     s.Protocol.Value,
			InternalPort: s.InternalPort.Value,
		})
	}
	return services
}

func ServicesToTfServices(input []apiv1.Service) []TfService {
	tfservices := make([]TfService, 0)
	for _, s := range input {
		var tfports []TfPort
		for _, j := range s.Ports {
			var handlers []types.String
			for _, k := range j.Handlers {
				handlers = append(handlers, types.String{Value: k})
			}
			tfports = append(tfports, TfPort{
				Port:     types.Int64{Value: j.Port},
				Handlers: handlers,
			})
		}
		tfservices = append(tfservices, TfService{
			Ports:        tfports,
			Protocol:     types.String{Value: s.Protocol},
			InternalPort: types.Int64{Value: s.InternalPort},
		})
	}
	return tfservices
}

func (mr flyMachineResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	_, err := mr.ValidateOpenTunnel()
	if err != nil {
		resp.Diagnostics.AddError("fly wireguard tunnel must be open", err.Error())
		return
	}

	var data flyMachineResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	services := TfServicesToServices(data.Services)
	createReq := apiv1.MachineCreateOrUpdateRequest{
		Name:   data.Name.Value,
		Region: data.Region.Value,
		Config: apiv1.MachineConfig{
			Image:    data.Image.Value,
			Services: services,
			Init: apiv1.InitConfig{
				Cmd:        data.Cmd,
				Entrypoint: data.Entrypoint,
			},
		},
	}

	if !data.Cpus.Unknown {
		createReq.Guest.Cpus = int(data.Cpus.Value)
	}
	if !data.CpuType.Unknown {
		createReq.Guest.CpuType = data.CpuType.Value
	}
	if !data.MemoryMb.Unknown {
		createReq.Guest.MemoryMb = int(data.MemoryMb.Value)
	}

	if !data.Env.Unknown {
		var env map[string]string
		data.Env.ElementsAs(context.Background(), &env, false)
		createReq.Config.Env = env
	}
	if len(data.Mounts) > 0 {
		var mounts []apiv1.MachineMount
		for _, m := range data.Mounts {
			mounts = append(mounts, apiv1.MachineMount{
				Encrypted: m.Encrypted.Value,
				Path:      m.Path.Value,
				SizeGb:    int(m.SizeGb.Value),
				Volume:    m.Volume.Value,
			})
		}
		createReq.Config.Mounts = mounts
	}

	machineAPI := apiv1.NewMachineAPI(mr.provider.httpClient, mr.provider.httpEndpoint)

	var newMachine apiv1.MachineResponse
	err = machineAPI.CreateMachine(createReq, data.App.Value, &newMachine)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create machine", err.Error())
		return
	}

	tflog.Info(ctx, fmt.Sprintf("%+v", newMachine))

	env := KVToTfMap(newMachine.Config.Env, types.StringType)

	tfservices := ServicesToTfServices(newMachine.Config.Services)

	if data.Services == nil {
		tfservices = nil
	}

	data = flyMachineResourceData{
		Name:       types.String{Value: newMachine.Name},
		Region:     types.String{Value: newMachine.Region},
		Id:         types.String{Value: newMachine.ID},
		App:        types.String{Value: data.App.Value},
		Image:      types.String{Value: newMachine.Config.Image},
		Cpus:       types.Int64{Value: int64(newMachine.Config.Guest.Cpus)},
		MemoryMb:   types.Int64{Value: int64(newMachine.Config.Guest.MemoryMb)},
		CpuType:    types.String{Value: newMachine.Config.Guest.CPUKind},
		Cmd:        newMachine.Config.Init.Cmd,
		Entrypoint: newMachine.Config.Init.Entrypoint,
		Env:        env,
		Services:   tfservices,
	}

	//if newMachine.Config.Init.Entrypoint != nil {
	//	tflog.Info(ctx, "WE SHOULD NOT BE HERE")
	//	data.Cmd = types.String{Value: *newMachine.Config.Init.Entrypoint}
	//}
	//if newMachine.Config.Init.Cmd != nil {
	//	tflog.Info(ctx, "WE SHOULD NOT BE HERE")
	//	data.Cmd = types.String{Value: *newMachine.Config.Init.Cmd}
	//}

	if len(newMachine.Config.Mounts) > 0 {
		var tfmounts []TfMachineMount
		for _, m := range newMachine.Config.Mounts {
			tfmounts = append(tfmounts, TfMachineMount{
				Encrypted: types.Bool{Value: m.Encrypted},
				Path:      types.String{Value: m.Path},
				SizeGb:    types.Int64{Value: int64(m.SizeGb)},
				Volume:    types.String{Value: m.Volume},
			})
		}
		data.Mounts = tfmounts
	}

	_, err = mr.provider.httpClient.R().Get(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s/wait?instance_id=%s", mr.provider.httpEndpoint, data.App.Value, data.Id.Value, newMachine.InstanceID))
	if err != nil {
		//FIXME(?): For now we just assume that the orchestrator is in fact going to faithfully execute our request
		tflog.Info(ctx, "Waiting errored")
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (mr flyMachineResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	_, err := mr.ValidateOpenTunnel()
	if err != nil {
		resp.Diagnostics.AddError("fly wireguard tunnel must be open", err.Error())
	}

	var data flyMachineResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	machineAPI := apiv1.NewMachineAPI(mr.provider.httpClient, mr.provider.httpEndpoint)

	var machine apiv1.MachineResponse

	_, err = machineAPI.ReadMachine(data.App.Value, data.Id.Value, &machine)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create machine", err.Error())
		return
	}

	env := KVToTfMap(machine.Config.Env, types.StringType)

	tfservices := ServicesToTfServices(machine.Config.Services)

	if data.Services == nil {
		tfservices = nil
	}

	data = flyMachineResourceData{
		Name:       types.String{Value: machine.Name},
		Id:         types.String{Value: machine.ID},
		Region:     types.String{Value: machine.Region},
		App:        types.String{Value: data.App.Value},
		Image:      types.String{Value: machine.Config.Image},
		Cpus:       types.Int64{Value: int64(machine.Config.Guest.Cpus)},
		MemoryMb:   types.Int64{Value: int64(machine.Config.Guest.MemoryMb)},
		CpuType:    types.String{Value: machine.Config.Guest.CPUKind},
		Cmd:        machine.Config.Init.Cmd,
		Entrypoint: machine.Config.Init.Entrypoint,
		Env:        env,
		Services:   tfservices,
	}

	if len(machine.Config.Mounts) > 0 {
		var tfmounts []TfMachineMount
		for _, m := range machine.Config.Mounts {
			tfmounts = append(tfmounts, TfMachineMount{
				Encrypted: types.Bool{Value: m.Encrypted},
				Path:      types.String{Value: m.Path},
				SizeGb:    types.Int64{Value: int64(m.SizeGb)},
				Volume:    types.String{Value: m.Volume},
			})
		}
		data.Mounts = tfmounts
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (mr flyMachineResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	_, err := mr.ValidateOpenTunnel()
	if err != nil {
		resp.Diagnostics.AddError("fly wireguard tunnel must be open", err.Error())
		return
	}

	var plan flyMachineResourceData

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var state flyMachineResourceData

	diags = resp.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Name.Unknown && plan.Name.Value != state.Name.Value {
		resp.Diagnostics.AddError("Can't mutate name of existing machine", "Can't swith name "+state.Name.Value+" to "+plan.Name.Value)
	}
	if !state.Region.Unknown && plan.Region.Value != state.Region.Value {
		resp.Diagnostics.AddError("Can't mutate region of existing machine", "Can't swith region "+state.Name.Value+" to "+plan.Name.Value)
	}

	services := TfServicesToServices(plan.Services)

	updateReq := apiv1.MachineCreateOrUpdateRequest{
		Name:   plan.Name.Value,
		Region: state.Region.Value,
		Config: apiv1.MachineConfig{
			Image:    plan.Image.Value,
			Services: services,
			Init: apiv1.InitConfig{
				Cmd:        plan.Cmd,
				Entrypoint: plan.Entrypoint,
			},
		},
	}

	if !plan.Cpus.Unknown {
		updateReq.Guest.Cpus = int(plan.Cpus.Value)
	}
	if !plan.CpuType.Unknown {
		updateReq.Guest.CpuType = plan.CpuType.Value
	}
	if !plan.MemoryMb.Unknown {
		updateReq.Guest.MemoryMb = int(plan.MemoryMb.Value)
	}
	if plan.Env.Null {
		env := map[string]string{}
		updateReq.Config.Env = env
	} else if !plan.Env.Unknown {
		var env map[string]string
		plan.Env.ElementsAs(context.Background(), &env, false)
		updateReq.Config.Env = env
	} else if !state.Env.Unknown {
		updateReq.Config.Env = map[string]string{}
	}

	if len(plan.Mounts) > 0 {
		var mounts []apiv1.MachineMount
		for _, m := range plan.Mounts {
			mounts = append(mounts, apiv1.MachineMount{
				Encrypted: m.Encrypted.Value,
				Path:      m.Path.Value,
				SizeGb:    int(m.SizeGb.Value),
				Volume:    m.Volume.Value,
			})
		}
		updateReq.Config.Mounts = mounts
	}

	machineApi := apiv1.NewMachineAPI(mr.provider.httpClient, mr.provider.httpEndpoint)

	var updatedMachine apiv1.MachineResponse

	createResponse, err := machineApi.UpdateMachine(updateReq, state.App.Value, state.Id.Value, &updatedMachine)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create machine", err.Error())
		return
	}

	if createResponse.StatusCode != http.StatusCreated && createResponse.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError("Update request failed", fmt.Sprintf("%s, %+v", createResponse.Status, updatedMachine))
		return
	}

	env := KVToTfMap(updatedMachine.Config.Env, types.StringType)

	tfservices := ServicesToTfServices(updatedMachine.Config.Services)

	if state.Services == nil {
		tfservices = nil
	}

	state = flyMachineResourceData{
		Name:       types.String{Value: updatedMachine.Name},
		Region:     types.String{Value: updatedMachine.Region},
		Id:         types.String{Value: updatedMachine.ID},
		App:        types.String{Value: state.App.Value},
		Image:      types.String{Value: updatedMachine.Config.Image},
		Cpus:       types.Int64{Value: int64(updatedMachine.Config.Guest.Cpus)},
		MemoryMb:   types.Int64{Value: int64(updatedMachine.Config.Guest.MemoryMb)},
		CpuType:    types.String{Value: updatedMachine.Config.Guest.CPUKind},
		Cmd:        updatedMachine.Config.Init.Cmd,
		Entrypoint: updatedMachine.Config.Init.Entrypoint,
		Env:        env,
		Services:   tfservices,
	}

	if len(updatedMachine.Config.Mounts) > 0 {
		var tfmounts []TfMachineMount
		for _, m := range updatedMachine.Config.Mounts {
			tfmounts = append(tfmounts, TfMachineMount{
				Encrypted: types.Bool{Value: m.Encrypted},
				Path:      types.String{Value: m.Path},
				SizeGb:    types.Int64{Value: int64(m.SizeGb)},
				Volume:    types.String{Value: m.Volume},
			})
		}
		state.Mounts = tfmounts
	}

	_, err = mr.provider.httpClient.R().Get(fmt.Sprintf("http://%s/v1/apps/%s/machines/%s/wait?instance_id=%s", mr.provider.httpEndpoint, state.App.Value, state.Id.Value, updatedMachine.InstanceID))
	if err != nil {
		tflog.Info(ctx, "Waiting errored")
	}

	resp.State.Set(ctx, state)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (mr flyMachineResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data flyMachineResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	_, err := mr.ValidateOpenTunnel()
	if err != nil {
		resp.Diagnostics.AddError("fly wireguard tunnel must be open", err.Error())
	}

	machineApi := apiv1.NewMachineAPI(mr.provider.httpClient, mr.provider.httpEndpoint)

	err = machineApi.DeleteMachine(data.App.Value, data.Id.Value, 50)

	if err != nil {
		resp.Diagnostics.AddError("Machine delete failed", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (mr flyMachineResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

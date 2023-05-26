package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/fly-apps/terraform-provider-fly/pkg/apiv1"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &flyMachineResource{}
var _ resource.ResourceWithConfigure = &flyMachineResource{}
var _ resource.ResourceWithImportState = &flyMachineResource{}


type flyMachineResource struct {
    config ProviderConfig
}

func NewMachineResource() resource.Resource {
	return &flyMachineResource{}
}

func (r *flyMachineResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "fly_machine"
}

func (r *flyMachineResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
    r.config = req.ProviderData.(ProviderConfig)
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
	PrivateIP  types.String `tfsdk:"privateip"`
	App        types.String `tfsdk:"app"`
	Image      types.String `tfsdk:"image"`
	Cpus       types.Int64  `tfsdk:"cpus"`
	MemoryMb   types.Int64  `tfsdk:"memorymb"`
	CpuType    types.String `tfsdk:"cputype"`
	Env        types.Map    `tfsdk:"env"`
	Cmd        []string     `tfsdk:"cmd"`
	Entrypoint []string     `tfsdk:"entrypoint"`
	Exec       []string     `tfsdk:"exec"`

	Mounts   []TfMachineMount `tfsdk:"mounts"`
	Services []TfService      `tfsdk:"services"`
}

type TfMachineMount struct {
	Encrypted types.Bool   `tfsdk:"encrypted"`
	Path      types.String `tfsdk:"path"`
	SizeGb    types.Int64  `tfsdk:"size_gb"`
	Volume    types.String `tfsdk:"volume"`
}

func (r *flyMachineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse){
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fly machine resource",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "machine name",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "machine region",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "machine id",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app": schema.StringAttribute{
				MarkdownDescription: "fly app",
				Required:            true,
			},
			"privateip": schema.StringAttribute{
				MarkdownDescription: "Private IP",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cmd": schema.ListAttribute{
				MarkdownDescription: "cmd",
				Optional:            true,
                ElementType: types.StringType,
			},
			"entrypoint": schema.ListAttribute{
				MarkdownDescription: "image entrypoint",
				Optional:            true,
                ElementType: types.StringType,
			},
			"exec": schema.ListAttribute{
				MarkdownDescription: "exec command",
				Optional:            true,
                ElementType: types.StringType,
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "docker image",
				Required:            true,
			},
			"cputype": schema.StringAttribute{
				MarkdownDescription: "cpu type",
				Computed:            true,
				Optional:            true,
			},
			"cpus": schema.Int64Attribute{
				MarkdownDescription: "cpu count",
				Computed:            true,
				Optional:            true,
			},
			"memorymb": schema.Int64Attribute{
				MarkdownDescription: "memory mb",
				Computed:            true,
				Optional:            true,
			},
			"env": schema.MapAttribute{
				MarkdownDescription: "Optional environment variables, keys and values must be strings",
				Optional:            true,
				Computed:            true,
                ElementType: types.StringType,
			},
			"mounts": schema.ListNestedAttribute{
				MarkdownDescription: "Volume mounts",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "encrypted": schema.BoolAttribute{
                            Optional: true,
                            Computed: true,
                        },
                        "path": schema.StringAttribute{
                            Required:            true,
                            MarkdownDescription: "Path for volume to be mounted on vm",
                        },
                        "size_gb": schema.Int64Attribute{
                            Optional: true,
                            Computed: true,
                        },
                        "volume": schema.StringAttribute{
                            Required:            true,
                            MarkdownDescription: "Name or ID of volume",
                        },
                    },
                },
			},
			"services": schema.ListNestedAttribute{
				MarkdownDescription: "services",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
                    Attributes: map[string]schema.Attribute{
                        "ports": schema.ListNestedAttribute{
                            MarkdownDescription: "External ports and handlers",
                            Required:            true,
                            NestedObject: schema.NestedAttributeObject {
                                Attributes: map[string]schema.Attribute{
                                    "port": schema.Int64Attribute{
                                        MarkdownDescription: "External port",
                                        Required:            true,
                                    },
                                    "handlers": schema.ListAttribute{
                                        MarkdownDescription: "How the edge should process requests",
                                        Optional:            true,
                                        ElementType: types.StringType,
                                    },
                                },
                            },
                        },
                        "protocol": schema.StringAttribute{
                            MarkdownDescription: "network protocol",
                            Required:            true,
                        },
                        "internal_port": schema.Int64Attribute{
                            MarkdownDescription: "Port application listens on internally",
                            Required:            true,
                        },
                    },
                },
			},
		},
	}
}

/// todo
func (r *flyMachineResource) ValidateOpenTunnel() (bool, error) {
	_, err := r.config.httpClient.R().Get(fmt.Sprintf("http://%s", r.config.httpEndpoint))
	if err != nil {
		return false, errors.New("can't connect to the api, is the tunnel open? :)")
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
				handlers = append(handlers, k.ValueString())
			}
			ports = append(ports, apiv1.Port{
				Port:     j.Port.ValueInt64(),
				Handlers: handlers,
			})
		}
		services = append(services, apiv1.Service{
			Ports:        ports,
			Protocol:     s.Protocol.ValueString(),
			InternalPort: s.InternalPort.ValueInt64(),
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
				handlers = append(handlers, types.StringValue(k))
			}
			tfports = append(tfports, TfPort{
				Port:     types.Int64Value(j.Port),
				Handlers: handlers,
			})
		}
		tfservices = append(tfservices, TfService{
			Ports:        tfports,
			Protocol:     types.StringValue(s.Protocol),
			InternalPort: types.Int64Value(s.InternalPort),
		})
	}
	return tfservices
}

func (r *flyMachineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	_, err := r.ValidateOpenTunnel()
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
		Name:   data.Name.ValueString(),
		Region: data.Region.ValueString(),
		Config: apiv1.MachineConfig{
			Image:    data.Image.ValueString(),
			Services: services,
			Init: apiv1.InitConfig{
				Cmd:        data.Cmd,
				Entrypoint: data.Entrypoint,
				Exec:       data.Exec,
			},
		},
	}

	if !data.Cpus.IsUnknown() {
		createReq.Config.Guest.Cpus = int(data.Cpus.ValueInt64())
	}
	if !data.CpuType.IsUnknown() {
		createReq.Config.Guest.CpuType = data.CpuType.ValueString()
	}
	if !data.MemoryMb.IsUnknown() {
		createReq.Config.Guest.MemoryMb = int(data.MemoryMb.ValueInt64())
	}

	if !data.Env.IsUnknown() {
		var env map[string]string
		data.Env.ElementsAs(context.Background(), &env, false)
		createReq.Config.Env = env
	}
	if len(data.Mounts) > 0 {
		var mounts []apiv1.MachineMount
		for _, m := range data.Mounts {
			mounts = append(mounts, apiv1.MachineMount{
				Encrypted: m.Encrypted.ValueBool(),
				Path:      m.Path.ValueString(),
				SizeGb:    int(m.SizeGb.ValueInt64()),
				Volume:    m.Volume.ValueString(),
			})
		}
		createReq.Config.Mounts = mounts
	}

	machineAPI := apiv1.NewMachineAPI(r.config.httpClient, r.config.httpEndpoint)

	var newMachine apiv1.MachineResponse
	err = machineAPI.CreateMachine(createReq, data.App.ValueString(), &newMachine)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create machine", err.Error())
		return
	}

	tflog.Info(ctx, fmt.Sprintf("%+v", newMachine))

	// env := utils.KVToTfMap(newMachine.Config.Env, types.StringType)
    env, diags := types.MapValueFrom(ctx, types.StringType, newMachine.Config.Env)
    resp.Diagnostics.Append(diags...)


	tfservices := ServicesToTfServices(newMachine.Config.Services)

	if data.Services == nil && len(tfservices) == 0 {
		tfservices = nil
	}

	data = flyMachineResourceData{
		Name:       types.StringValue(newMachine.Name),
		Region:     types.StringValue(newMachine.Region),
		Id:         types.StringValue(newMachine.ID),
		App:        data.App,
		PrivateIP:  types.StringValue(newMachine.PrivateIP),
		Image:      types.StringValue(newMachine.Config.Image),
		Cpus:       types.Int64Value(int64(newMachine.Config.Guest.Cpus)),
		MemoryMb:   types.Int64Value(int64(newMachine.Config.Guest.MemoryMb)),
		CpuType:    types.StringValue(newMachine.Config.Guest.CPUKind),
		Cmd:        newMachine.Config.Init.Cmd,
		Entrypoint: newMachine.Config.Init.Entrypoint,
		Exec:       newMachine.Config.Init.Exec,
		Env:        env,
		Services:   tfservices,
	}

	if len(newMachine.Config.Mounts) > 0 {
		var tfmounts []TfMachineMount
		for _, m := range newMachine.Config.Mounts {
			tfmounts = append(tfmounts, TfMachineMount{
				Encrypted: types.BoolValue(m.Encrypted),
				Path:      types.StringValue(m.Path),
				SizeGb:    types.Int64Value(int64(m.SizeGb)),
				Volume:    types.StringValue(m.Volume),
			})
		}
		data.Mounts = tfmounts
	}

	err = machineAPI.WaitForMachine(data.App.ValueString(), data.Id.ValueString(), newMachine.InstanceID)
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

func (r *flyMachineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	_, err := r.ValidateOpenTunnel()
	if err != nil {
		resp.Diagnostics.AddError("fly wireguard tunnel must be open", err.Error())
	}

	var data flyMachineResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	machineAPI := apiv1.NewMachineAPI(r.config.httpClient, r.config.httpEndpoint)

	var machine apiv1.MachineResponse

	_, err = machineAPI.ReadMachine(data.App.ValueString(), data.Id.ValueString(), &machine)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create machine", err.Error())
		return
	}

	// env := utils.KVToTfMap(machine.Config.Env, types.StringType)
    env, diags := types.MapValueFrom(ctx, types.StringType, machine.Config.Env)
    resp.Diagnostics.Append(diags...)

	tfservices := ServicesToTfServices(machine.Config.Services)

	if data.Services == nil && len(tfservices) == 0 {
		tfservices = nil
	}

	data = flyMachineResourceData{
		Name:       types.StringValue(machine.Name),
		Id:         types.StringValue(machine.ID),
		Region:     types.StringValue(machine.Region),
		App:        data.App,
		PrivateIP:  types.StringValue(machine.PrivateIP),
		Image:      types.StringValue(machine.Config.Image),
		Cpus:       types.Int64Value(int64(machine.Config.Guest.Cpus)),
		MemoryMb:   types.Int64Value(int64(machine.Config.Guest.MemoryMb)),
		CpuType:    types.StringValue(machine.Config.Guest.CPUKind),
		Cmd:        machine.Config.Init.Cmd,
		Entrypoint: machine.Config.Init.Entrypoint,
		Exec:       machine.Config.Init.Exec,
		Env:        env,
		Services:   tfservices,
	}

	if len(machine.Config.Mounts) > 0 {
		var tfmounts []TfMachineMount
		for _, m := range machine.Config.Mounts {
			tfmounts = append(tfmounts, TfMachineMount{
				Encrypted: types.BoolValue(m.Encrypted),
				Path:      types.StringValue(m.Path),
				SizeGb:    types.Int64Value(int64(m.SizeGb)),
				Volume:    types.StringValue(m.Volume),
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

func (r *flyMachineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	_, err := r.ValidateOpenTunnel()
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

	if !plan.Name.IsUnknown() && plan.Name.ValueString() != state.Name.ValueString() {
		resp.Diagnostics.AddError("Can't mutate name of existing machine", "Can't switch name "+state.Name.ValueString()+" to "+plan.Name.ValueString())
	}
	if !state.Region.IsUnknown() && plan.Region.ValueString() != state.Region.ValueString() {
		resp.Diagnostics.AddError("Can't mutate region of existing machine", "Can't switch region "+state.Name.ValueString()+" to "+plan.Name.ValueString())
	}

	services := TfServicesToServices(plan.Services)

	updateReq := apiv1.MachineCreateOrUpdateRequest{
		Name:   plan.Name.ValueString(),
		Region: state.Region.ValueString(),
		Config: apiv1.MachineConfig{
			Image:    plan.Image.ValueString(),
			Services: services,
			Init: apiv1.InitConfig{
				Cmd:        plan.Cmd,
				Entrypoint: plan.Entrypoint,
				Exec:       plan.Exec,
			},
		},
	}

	if !plan.Cpus.IsUnknown() {
		updateReq.Config.Guest.Cpus = int(plan.Cpus.ValueInt64())
	}
	if !plan.CpuType.IsUnknown() {
		updateReq.Config.Guest.CpuType = plan.CpuType.ValueString()
	}
	if !plan.MemoryMb.IsUnknown() {
		updateReq.Config.Guest.MemoryMb = int(plan.MemoryMb.ValueInt64())
	}
	if plan.Env.IsNull() {
		env := map[string]string{}
		updateReq.Config.Env = env
	} else if !plan.Env.IsUnknown() {
		var env map[string]string
		plan.Env.ElementsAs(context.Background(), &env, false)
		updateReq.Config.Env = env
	} else if !state.Env.IsUnknown() {
		updateReq.Config.Env = map[string]string{}
	}

	if len(plan.Mounts) > 0 {
		var mounts []apiv1.MachineMount
		for _, m := range plan.Mounts {
			mounts = append(mounts, apiv1.MachineMount{
				Encrypted: m.Encrypted.ValueBool(),
				Path:      m.Path.ValueString(),
				SizeGb:    int(m.SizeGb.ValueInt64()),
				Volume:    m.Volume.ValueString(),
			})
		}
		updateReq.Config.Mounts = mounts
	}

	machineApi := apiv1.NewMachineAPI(r.config.httpClient, r.config.httpEndpoint)

	var updatedMachine apiv1.MachineResponse

	err = machineApi.UpdateMachine(updateReq, state.App.ValueString(), state.Id.ValueString(), &updatedMachine)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update machine", err.Error())
		return
	}

	// env := utils.KVToTfMap(updatedMachine.Config.Env, types.StringType)
    env, diags := types.MapValueFrom(ctx, types.StringType, updatedMachine.Config.Env)
    resp.Diagnostics.Append(diags...)

	tfservices := ServicesToTfServices(updatedMachine.Config.Services)

	state = flyMachineResourceData{
		Name:       types.StringValue(updatedMachine.Name),
		Region:     types.StringValue(updatedMachine.Region),
		Id:         types.StringValue(updatedMachine.ID),
		App:        state.App,
		PrivateIP:  types.StringValue(updatedMachine.PrivateIP),
		Image:      types.StringValue(updatedMachine.Config.Image),
		Cpus:       types.Int64Value(int64(updatedMachine.Config.Guest.Cpus)),
		MemoryMb:   types.Int64Value(int64(updatedMachine.Config.Guest.MemoryMb)),
		CpuType:    types.StringValue(updatedMachine.Config.Guest.CPUKind),
		Cmd:        updatedMachine.Config.Init.Cmd,
		Entrypoint: updatedMachine.Config.Init.Entrypoint,
		Exec:       updatedMachine.Config.Init.Exec,
		Env:        env,
		Services:   tfservices,
	}

	if len(updatedMachine.Config.Mounts) > 0 {
		var tfmounts []TfMachineMount
		for _, m := range updatedMachine.Config.Mounts {
			tfmounts = append(tfmounts, TfMachineMount{
				Encrypted: types.BoolValue(m.Encrypted),
				Path:      types.StringValue(m.Path),
				SizeGb:    types.Int64Value(int64(m.SizeGb)),
				Volume:    types.StringValue(m.Volume),
			})
		}
		state.Mounts = tfmounts
	}

	err = machineApi.WaitForMachine(state.App.ValueString(), state.Id.ValueString(), updatedMachine.InstanceID)
	if err != nil {
		tflog.Info(ctx, "Waiting errored")
	}

	resp.State.Set(ctx, state)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *flyMachineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data flyMachineResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	_, err := r.ValidateOpenTunnel()
	if err != nil {
		resp.Diagnostics.AddError("fly wireguard tunnel must be open", err.Error())
	}

	machineApi := apiv1.NewMachineAPI(r.config.httpClient, r.config.httpEndpoint)

	err = machineApi.DeleteMachine(data.App.ValueString(), data.Id.ValueString(), 50)

	if err != nil {
		resp.Diagnostics.AddError("Machine delete failed", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (mr flyMachineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: app_id,machine_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
}

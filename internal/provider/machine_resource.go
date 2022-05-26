package provider

import (
	"bytes"
	"context"
	"dov.dev/fly/fly-provider/internal/utils"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"io/ioutil"
	"net/http"
	"time"
)

var _ tfsdk.ResourceType = flyMachineResourceType{}
var _ tfsdk.Resource = flyMachineResource{}
var _ tfsdk.ResourceWithImportState = flyMachineResource{}

//TODO: build

type flyMachineResourceType struct {
	Token string
}

type flyMachineResource struct {
	provider provider
	http     http.Client
}

type GuestConfig struct {
	Cpus     int    `json:"cpus,omitempty"`
	MemoryMb int    `json:"memory_mb,omitempty"`
	CpuType  string `json:"cpu_type,omitempty"`
}

type Port struct {
	Port     int64    `json:"port" tfsdk:"port"`
	Handlers []string `json:"handlers" tfsdk:"handlers"`
}

type TfPort struct {
	Port     types.Int64    `tfsdk:"port"`
	Handlers []types.String `tfsdk:"handlers"`
}

type Service struct {
	Ports        []Port `json:"ports" tfsdk:"ports"`
	Protocol     string `json:"protocol" tfsdk:"protocol"`
	InternalPort int64  `json:"internal_port" tfsdk:"internal_port"`
}

type TfService struct {
	Ports        []TfPort     `tfsdk:"ports"`
	Protocol     types.String `tfsdk:"protocol"`
	InternalPort types.Int64  `tfsdk:"internal_port"`
}

type MachineConfig struct {
	Image    string            `json:"image"`
	Env      map[string]string `json:"env,omitempty"`
	Services []Service
}

type CreateOrUpdateMachineRequest struct {
	Name   string        `json:"name"`
	Config MachineConfig `json:"config"`
	Guest  *GuestConfig  `json:"guest,omitempty"`
}

// Api response

type MachineResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	State      string `json:"state"`
	Region     string `json:"region"`
	InstanceID string `json:"instance_id"`
	PrivateIP  string `json:"private_ip"`
	Config     struct {
		Env  map[string]string `json:"env"`
		Init struct {
			Exec       interface{} `json:"exec"`
			Entrypoint interface{} `json:"entrypoint"`
			Cmd        interface{} `json:"cmd"`
			Tty        bool        `json:"tty"`
		} `json:"init"`
		Image    string      `json:"image"`
		Metadata interface{} `json:"metadata"`
		Restart  struct {
			Policy string `json:"policy"`
		} `json:"restart"`
		Services []struct {
			InternalPort int    `json:"internal_port"`
			Ports        []Port `json:"ports"`
			Protocol     string `json:"protocol"`
		} `json:"services"`
		Guest struct {
			CPUKind  string `json:"cpu_kind"`
			Cpus     int    `json:"cpus"`
			MemoryMb int    `json:"memory_mb"`
		} `json:"guest"`
	} `json:"config"`
	ImageRef struct {
		Registry   string `json:"registry"`
		Repository string `json:"repository"`
		Tag        string `json:"tag"`
		Digest     string `json:"digest"`
		Labels     struct {
		} `json:"labels"`
	} `json:"image_ref"`
	CreatedAt time.Time `json:"created_at"`
}

type flyMachineResourceData struct {
	Name     types.String `tfsdk:"name"`
	Region   types.String `tfsdk:"region"`
	Id       types.String `tfsdk:"id"`
	App      types.String `tfsdk:"app"`
	Image    types.String `tfsdk:"image"`
	Cpus     types.Int64  `tfsdk:"cpus"`
	MemoryMb types.Int64  `tfsdk:"memorymb"`
	CpuType  types.String `tfsdk:"cputype"`
	Env      types.Map    `tfsdk:"env"`

	Services []TfService `tfsdk:"services"`
}

func (mr flyMachineResourceType) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Fly machine resource",
		Attributes: map[string]tfsdk.Attribute{
			"name": {
				MarkdownDescription: "machine name",
				Required:            true,
				Type:                types.StringType,
			},
			"region": {
				MarkdownDescription: "machine region",
				Required:            true,
				Type:                types.StringType,
			},
			"id": {
				MarkdownDescription: "machine id",
				Computed:            true,
				Type:                types.StringType,
			},
			"app": {
				MarkdownDescription: "fly app",
				Required:            true,
				Type:                types.StringType,
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
			"services": {
				MarkdownDescription: "services",
				Required:            true,
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
						}, tfsdk.ListNestedAttributesOptions{}),
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
				}, tfsdk.ListNestedAttributesOptions{}),
			},
		},
	}, nil
}

func (mr flyMachineResourceType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	h := http.Client{Timeout: 60 * time.Second, Transport: &utils.Transport{UnderlyingTransport: http.DefaultTransport, Token: mr.Token, Ctx: ctx}}
	return flyMachineResource{
		provider: provider,
		http:     h,
	}, diags
}

func (mr flyMachineResource) ValidateOpenTunnel() (bool, error) {
	//HACK: This is not a good way to do this, but I'm tired. Future me, please fix this.
	response, err := mr.http.Get("http://127.0.0.1:4280/bogus")
	if err != nil {
		return false, err
	}
	if response.Status == "404 Not Found" {
		return true, nil
	} else {
		return false, errors.New("unexpected in ValidateOpenTunnel. File an issue")
	}
}

func (mr flyMachineResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data flyMachineResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	_, err := mr.ValidateOpenTunnel()
	if err != nil {
		resp.Diagnostics.AddError("fly wireguard tunnel must be open", err.Error())
		return
	}

	var services []Service
	for _, s := range data.Services {
		var ports []Port
		for _, s := range s.Ports {
			var handlers []string
			for _, k := range s.Handlers {
				handlers = append(handlers, k.Value)
			}
			ports = append(ports, Port{
				Port:     s.Port.Value,
				Handlers: handlers,
			})
		}
		services = append(services, Service{
			Ports:        ports,
			Protocol:     s.Protocol.Value,
			InternalPort: s.InternalPort.Value,
		})
	}

	createReq := CreateOrUpdateMachineRequest{
		Name: data.Name.Value,
		Config: MachineConfig{
			Image:    data.Image.Value,
			Services: services,
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

	body, _ := json.Marshal(createReq)
	var prettyJSON bytes.Buffer
	_ = json.Indent(&prettyJSON, body, "", "\t")
	createResponse, err := mr.http.Post(fmt.Sprintf("http://127.0.0.1:4280/v1/apps/%s/machines", data.App.Value), "application/json", bytes.NewBuffer(body))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create machine", err.Error())
		return
	}

	defer createResponse.Body.Close()

	var newMachine MachineResponse
	if createResponse.StatusCode == http.StatusCreated || createResponse.StatusCode == http.StatusOK {
		err := json.NewDecoder(createResponse.Body).Decode(&newMachine)
		if err != nil {
			resp.Diagnostics.AddError("Failed to decode response machine", err.Error())
			return
		}
	} else {
		mp := make(map[string]interface{})
		_ = json.NewDecoder(createResponse.Body).Decode(&mp)
		resp.Diagnostics.AddError("Request failed", fmt.Sprintf("%s, %s, %+v", createResponse.Status, createResponse.Request.RequestURI, mp))
		return
	}

	var env types.Map
	env.ElemType = types.StringType
	for key, value := range newMachine.Config.Env {
		if env.Elems == nil {
			env.Elems = map[string]attr.Value{}
		}
		env.Elems[key] = types.String{Value: value}
	}

	var tfservices []TfService
	for _, s := range newMachine.Config.Services {
		var tfports []TfPort
		for _, s := range s.Ports {
			var handlers []types.String
			for _, k := range s.Handlers {
				handlers = append(handlers, types.String{Value: k})
			}
			tfports = append(tfports, TfPort{
				Port:     types.Int64{Value: s.Port},
				Handlers: handlers,
			})
		}
		tfservices = append(tfservices, TfService{
			Ports:        tfports,
			Protocol:     types.String{Value: s.Protocol},
			InternalPort: types.Int64{Value: int64(s.InternalPort)},
		})
	}

	data = flyMachineResourceData{
		Name:     types.String{Value: newMachine.Name},
		Region:   types.String{Value: newMachine.Region},
		Id:       types.String{Value: newMachine.ID},
		App:      types.String{Value: data.App.Value},
		Image:    types.String{Value: newMachine.Config.Image},
		Cpus:     types.Int64{Value: int64(newMachine.Config.Guest.Cpus)},
		MemoryMb: types.Int64{Value: int64(newMachine.Config.Guest.MemoryMb)},
		CpuType:  types.String{Value: newMachine.Config.Guest.CPUKind},
		Env:      env,
		Services: tfservices,
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (mr flyMachineResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data flyMachineResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	_, err := mr.ValidateOpenTunnel()
	if err != nil {
		resp.Diagnostics.AddError("fly wireguard tunnel must be open", err.Error())
	}

	readResponse, err := mr.http.Get(fmt.Sprintf("http://127.0.0.1:4280/v1/apps/%s/machines/%s", data.App.Value, data.Id.Value))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create machine", err.Error())
		return
	}
	defer readResponse.Body.Close()

	var machine MachineResponse
	if readResponse.StatusCode == http.StatusOK {
		body, err := ioutil.ReadAll(readResponse.Body)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read machine creation response", err.Error())
			return
		}
		err = json.Unmarshal(body, &machine)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read machine creation response", err.Error())
			return
		}
	} else {
		mp := make(map[string]interface{})
		_ = json.NewDecoder(readResponse.Body).Decode(&mp)
		resp.Diagnostics.AddError("Machine read request failed", fmt.Sprintf("%s, %s, %+v", readResponse.Status, readResponse.Request.RequestURI, mp))
		return
	}

	var env types.Map
	env.ElemType = types.StringType
	for key, value := range machine.Config.Env {
		if env.Elems == nil {
			env.Elems = map[string]attr.Value{}
		}
		env.Elems[key] = types.String{Value: value}
	}

	var tfservices []TfService
	for _, s := range machine.Config.Services {
		var tfports []TfPort
		for _, s := range s.Ports {
			var handlers []types.String
			for _, k := range s.Handlers {
				handlers = append(handlers, types.String{Value: k})
			}
			tfports = append(tfports, TfPort{
				Port:     types.Int64{Value: s.Port},
				Handlers: handlers,
			})
		}
		tfservices = append(tfservices, TfService{
			Ports:        tfports,
			Protocol:     types.String{Value: s.Protocol},
			InternalPort: types.Int64{Value: int64(s.InternalPort)},
		})
	}

	data = flyMachineResourceData{
		Name:     types.String{Value: machine.Name},
		Id:       types.String{Value: machine.ID},
		Region:   types.String{Value: machine.Region},
		App:      types.String{Value: data.App.Value},
		Image:    types.String{Value: machine.Config.Image},
		Cpus:     types.Int64{Value: int64(machine.Config.Guest.Cpus)},
		MemoryMb: types.Int64{Value: int64(machine.Config.Guest.MemoryMb)},
		CpuType:  types.String{Value: machine.Config.Guest.CPUKind},
		Env:      env,
		Services: tfservices,
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

	if !state.Name.Unknown && plan.Name.Value != state.Name.Value {
		resp.Diagnostics.AddError("Can't mutate name of existing machine", "Can't swith name "+state.Name.Value+" to "+plan.Name.Value)
	}
	if !state.Region.Unknown && plan.Region.Value != state.Region.Value {
		resp.Diagnostics.AddError("Can't mutate region of existing machine", "Can't swith region "+state.Name.Value+" to "+plan.Name.Value)
	}

	var services []Service
	for _, s := range plan.Services {
		var ports []Port
		for _, s := range s.Ports {
			var handlers []string
			for _, k := range s.Handlers {
				handlers = append(handlers, k.Value)
			}
			ports = append(ports, Port{
				Port:     s.Port.Value,
				Handlers: handlers,
			})
		}
		services = append(services, Service{
			Ports:        ports,
			Protocol:     s.Protocol.Value,
			InternalPort: s.InternalPort.Value,
		})
	}

	updateReq := CreateOrUpdateMachineRequest{
		Name: plan.Name.Value,
		Config: MachineConfig{
			Image:    plan.Image.Value,
			Services: services,
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
	if !plan.Env.Unknown {
		var env map[string]string
		plan.Env.ElementsAs(context.Background(), &env, false)
		updateReq.Config.Env = env
	}

	body, _ := json.Marshal(updateReq)
	var prettyJSON bytes.Buffer
	_ = json.Indent(&prettyJSON, body, "", "\t")
	updateResponse, err := mr.http.Post(fmt.Sprintf("http://127.0.0.1:4280/v1/apps/%s/machines/%s", state.App.Value, state.Id.Value), "application/json", bytes.NewBuffer(body))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create machine", err.Error())
		return
	}

	defer updateResponse.Body.Close()

	var updatedMachine MachineResponse
	if updateResponse.StatusCode == http.StatusCreated || updateResponse.StatusCode == http.StatusOK {
		err := json.NewDecoder(updateResponse.Body).Decode(&updatedMachine)
		if err != nil {
			resp.Diagnostics.AddError("Failed to decode response machine", err.Error())
			return
		}
	} else {
		mp := make(map[string]interface{})
		_ = json.NewDecoder(updateResponse.Body).Decode(&mp)
		resp.Diagnostics.AddError("Request failed", fmt.Sprintf("%s, %s, %+v", updateResponse.Status, updateResponse.Request.RequestURI, mp))
		return
	}

	var env types.Map
	env.ElemType = types.StringType
	for key, value := range updatedMachine.Config.Env {
		if env.Elems == nil {
			env.Elems = map[string]attr.Value{}
		}
		env.Elems[key] = types.String{Value: value}
	}

	var tfservices []TfService
	for _, s := range updatedMachine.Config.Services {
		var tfports []TfPort
		for _, s := range s.Ports {
			var handlers []types.String
			for _, k := range s.Handlers {
				handlers = append(handlers, types.String{Value: k})
			}
			tfports = append(tfports, TfPort{
				Port:     types.Int64{Value: s.Port},
				Handlers: handlers,
			})
		}
		tfservices = append(tfservices, TfService{
			Ports:        tfports,
			Protocol:     types.String{Value: s.Protocol},
			InternalPort: types.Int64{Value: int64(s.InternalPort)},
		})
	}

	state = flyMachineResourceData{
		Name:     types.String{Value: updatedMachine.Name},
		Region:   types.String{Value: updatedMachine.Region},
		Id:       types.String{Value: updatedMachine.ID},
		App:      types.String{Value: state.App.Value},
		Image:    types.String{Value: updatedMachine.Config.Image},
		Cpus:     types.Int64{Value: int64(updatedMachine.Config.Guest.Cpus)},
		MemoryMb: types.Int64{Value: int64(updatedMachine.Config.Guest.MemoryMb)},
		CpuType:  types.String{Value: updatedMachine.Config.Guest.CPUKind},
		Env:      env,
		Services: tfservices,
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

	maxRetries := 10
	deleted := false

	for i := 0; i < maxRetries; i++ {
		readResponse, err := mr.http.Get(fmt.Sprintf("http://127.0.0.1:4280/v1/apps/%s/machines/%s", data.App.Value, data.Id.Value))
		if err != nil {
			resp.Diagnostics.AddError("Failed to get machine", err.Error())
			return
		}
		if readResponse.StatusCode == 200 {
			var machine MachineResponse
			body, err := ioutil.ReadAll(readResponse.Body)
			if err != nil {
				resp.Diagnostics.AddError("Failed to read machine response", err.Error())
				return
			}
			err = json.Unmarshal(body, &machine)
			if err != nil {
				resp.Diagnostics.AddError("Failed to read machine response", err.Error())
				return
			}
			if machine.State == "started" {
				tflog.Info(ctx, "Stopping machine")
				_, _ = mr.http.Post(fmt.Sprintf("http://127.0.0.1:4280/v1/apps/%s/machines/%s/stop", data.App.Value, data.Id.Value), "application/json", nil)
			}
			if machine.State == "stopping" || machine.State == "destroying" {
				tflog.Info(ctx, "In the process of stopping or destroying")
				time.Sleep(5 * time.Second)
			}
			if machine.State == "stopped" {
				tflog.Info(ctx, "Destroying")
				req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:4280/v1/apps/%s/machines/%s", data.App.Value, data.Id.Value), nil)
				if err != nil {
					resp.Diagnostics.AddError("Failed to create deletion request", err.Error())
					return
				}
				_, err = mr.http.Do(req)
				if err != nil {
					resp.Diagnostics.AddError("Failed to delete", err.Error())
					return
				}
			}
			if machine.State == "destroyed" {
				deleted = true
				break
			}
		}
	}

	if !deleted {
		resp.Diagnostics.AddError("Machine delete failed", "max retries exceeded")
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

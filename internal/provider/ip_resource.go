package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	basegql "github.com/Khan/genqlient/graphql"
	"github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

var _ resource.Resource = &flyIpResource{}
var _ resource.ResourceWithConfigure = &flyIpResource{}
var _ resource.ResourceWithImportState = &flyIpResource{}

type flyIpResource struct {
	client *basegql.Client
}

func NewIpResource() resource.Resource {
	return &flyIpResource{}
}

func (r *flyIpResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "fly_ip"
}

func (r *flyIpResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config := req.ProviderData.(ProviderConfig)
	r.client = config.gqclient
}

type flyIpResourceData struct {
	Id      types.String `tfsdk:"id"`
	Appid   types.String `tfsdk:"app"`
	Region  types.String `tfsdk:"region"`
	Address types.String `tfsdk:"address"`
	Type    types.String `tfsdk:"type"`
}

func (r *flyIpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fly ip resource",
		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				MarkdownDescription: "IP address",
				Computed:            true,
			},
			"app": schema.StringAttribute{
				MarkdownDescription: "Name of app to attach to",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of address",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "v4 or v6",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "region",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("global"),
			},
		},
	}
}

func (r *flyIpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data flyIpResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, fmt.Sprintf("%+v", data))

	q, err := graphql.AllocateIpAddress(ctx, *r.client, data.Appid.ValueString(), data.Region.ValueString(), graphql.IPAddressType(data.Type.ValueString()))
	tflog.Info(ctx, fmt.Sprintf("query res in create ip: %+v", q))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create ip addr", err.Error())
	}

	data = flyIpResourceData{
		Id:      types.StringValue(q.AllocateIpAddress.IpAddress.Id),
		Appid:   types.StringValue(data.Appid.ValueString()),
		Region:  types.StringValue(q.AllocateIpAddress.IpAddress.Region),
		Type:    types.StringValue(string(q.AllocateIpAddress.IpAddress.Type)),
		Address: types.StringValue(q.AllocateIpAddress.IpAddress.Address),
	}

	tflog.Info(ctx, fmt.Sprintf("%+v", data))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r flyIpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data flyIpResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	addr := data.Address.ValueString()
	app := data.Appid.ValueString()

	query, err := graphql.IpAddressQuery(ctx, *r.client, app, addr)
	tflog.Info(ctx, fmt.Sprintf("Query res: for %s %s %+v", app, addr, query))
	var errList gqlerror.List
	if errors.As(err, &errList) {
		for _, err := range errList {
			tflog.Info(ctx, "IN HERE")
			if err.Message == "Could not resolve " {
				return
			}
			resp.Diagnostics.AddError(err.Message, err.Path.String())
		}
	} else if err != nil {
		resp.Diagnostics.AddError("Read: query failed", err.Error())
	}

	data = flyIpResourceData{
		Id:      types.StringValue(query.App.IpAddress.Id),
		Appid:   types.StringValue(data.Appid.ValueString()),
		Region:  types.StringValue(query.App.IpAddress.Region),
		Type:    types.StringValue(string(query.App.IpAddress.Type)),
		Address: types.StringValue(query.App.IpAddress.Address),
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *flyIpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("The fly api does not allow updating ips once created", "Try deleting and then recreating the ip with new options")
	return
}

func (r *flyIpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data flyIpResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if !data.Id.IsUnknown() && !data.Id.IsNull() && data.Id.ValueString() != "" {
		_, err := graphql.ReleaseIpAddress(ctx, *r.client, data.Id.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Release ip failed", err.Error())
		}
	}

	resp.State.RemoveResource(ctx)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *flyIpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: app_id,ip_address. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("address"), idParts[1])...)
}

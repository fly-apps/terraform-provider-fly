package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfsdkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdkprovider.ResourceType = flyVolumeResourceType{}
var _ resource.Resource = flyVolumeResource{}
var _ resource.ResourceWithImportState = flyVolumeResource{}

type flyVolumeResourceType struct{}

type flyVolumeResource struct {
	provider provider
}

type flyVolumeResourceData struct {
	Id         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Size       types.Int64  `tfsdk:"size"`
	Appid      types.String `tfsdk:"app"`
	Region     types.String `tfsdk:"region"`
	Internalid types.String `tfsdk:"internalid"`
}

func (t flyVolumeResourceType) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Fly volume resource",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				MarkdownDescription: "ID of volume",
				Type:                types.StringType,
				Computed:            true,
				Optional:            true,
			},
			"app": {
				MarkdownDescription: "Name of app to attach to",
				Required:            true,
				Type:                types.StringType,
			},
			"size": {
				MarkdownDescription: "Size of volume in GB",
				Required:            true,
				Type:                types.Int64Type,
			},
			"name": {
				MarkdownDescription: "name",
				Type:                types.StringType,
				Required:            true,
			},
			"region": {
				MarkdownDescription: "region",
				Type:                types.StringType,
				Required:            true,
			},
			"internalid": {
				MarkdownDescription: "Internal ID",
				Type:                types.StringType,
				Computed:            true,
				Optional:            true,
			},
		},
	}, nil
}

func (t flyVolumeResourceType) NewResource(ctx context.Context, in tfsdkprovider.Provider) (resource.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return flyVolumeResource{
		provider: provider,
	}, diags
}

func (vr flyVolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data flyVolumeResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	q, err := graphql.CreateVolume(context.Background(), *vr.provider.client, data.Appid.Value, data.Name.Value, data.Region.Value, int(data.Size.Value))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create volume", err.Error())
	}

	data = flyVolumeResourceData{
		Id:         types.String{Value: q.CreateVolume.Volume.Id},
		Name:       types.String{Value: q.CreateVolume.Volume.Name},
		Size:       types.Int64{Value: int64(q.CreateVolume.Volume.SizeGb)},
		Appid:      types.String{Value: data.Appid.Value},
		Region:     types.String{Value: q.CreateVolume.Volume.Region},
		Internalid: types.String{Value: q.CreateVolume.Volume.InternalId},
	}

	tflog.Info(ctx, fmt.Sprintf("%+v", data))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (vr flyVolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data flyVolumeResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	internalId := data.Internalid.Value
	app := data.Appid.Value

	query, err := graphql.VolumeQuery(context.Background(), *vr.provider.client, app, internalId)
	if err != nil {
		resp.Diagnostics.AddError("Read: query failed", err.Error())
	}

	data = flyVolumeResourceData{
		Id:         types.String{Value: query.App.Volume.Id},
		Name:       types.String{Value: query.App.Volume.Name},
		Size:       types.Int64{Value: int64(query.App.Volume.SizeGb)},
		Appid:      types.String{Value: data.Appid.Value},
		Region:     types.String{Value: query.App.Volume.Region},
		Internalid: types.String{Value: query.App.Volume.InternalId},
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (vr flyVolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("The fly api does not allow updating volumes once created", "Try deleting and then recreating a volume with new options")
	return
}

func (vr flyVolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data flyVolumeResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if !data.Id.Unknown && !data.Id.Null && data.Id.Value != "" {
		_, err := graphql.DeleteVolume(context.Background(), *vr.provider.client, data.Id.Value)
		if err != nil {
			resp.Diagnostics.AddError("Delete volume failed", err.Error())
		}
	}

	resp.State.RemoveResource(ctx)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (vr flyVolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ",")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: app_id,volume_internal_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("app"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("internalid"), idParts[1])...)
}

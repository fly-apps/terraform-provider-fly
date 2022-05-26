package provider

import (
	"context"
	"dov.dev/fly/fly-provider/graphql"
	"dov.dev/fly/fly-provider/internal/provider/modifiers"
	"dov.dev/fly/fly-provider/internal/utils"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ tfsdk.ResourceType = flyPgResourceType{}
var _ tfsdk.Resource = flyPgResource{}
var _ tfsdk.ResourceWithImportState = flyPgResource{}

type flyPgResourceType struct{}

type flyPgResource struct {
	provider provider
}

type flyPgResourceData struct {
	//Id         types.String `tfsdk:"id"`
	Org        types.String `tfsdk:"org"`
	Name       types.String `tfsdk:"name"`
	Region     types.String `tfsdk:"region"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
	Vmsize     types.String `tfsdk:"vmsize"`
	Volumesize types.Int64  `tfsdk:"volumesize"`
	Count      types.Int64  `tfsdk:"count"`
}

func (t flyPgResourceType) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Fly postgres resource",
		Attributes: map[string]tfsdk.Attribute{
			"org": {
				MarkdownDescription: "Optional org ID to operate upon",
				Computed:            true,
				Optional:            true,
				Type:                types.StringType,
			},
			"name": {
				MarkdownDescription: "Name of cluster",
				Required:            true,
				Type:                types.StringType,
			},
			"region": {
				MarkdownDescription: "Starting region",
				Required:            true,
				Type:                types.StringType,
			},
			"username": {
				MarkdownDescription: "Database password",
				Computed:            true,
				Type:                types.StringType,
			},
			"password": {
				MarkdownDescription: "Database password",
				Optional:            true,
				Type:                types.StringType,
			},
			"vmsize": {
				MarkdownDescription: "Fly instance type (defaults to shared-cpu-1x)",
				Computed:            true,
				Optional:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					modifiers.StringDefault("shared-cpu-1x"),
				},
				Type: types.Int64Type,
			},
			"volumesize": {
				MarkdownDescription: "Persistent storage size",
				Optional:            true,
				Type:                types.Int64Type,
			},
		},
	}, nil
}

func (t flyPgResourceType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return flyPgResource{
		provider: provider,
	}, diags
}

func (r flyPgResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data flyPgResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if data.Org.Unknown {
		defaultOrg, err := utils.GetDefaultOrg(*r.provider.client)
		if err != nil {
			resp.Diagnostics.AddError("Could not detect default organization", err.Error())
			return
		}
		data.Org.Value = defaultOrg.Id
	}

	if data.Volumesize.Unknown {
		data.Volumesize.Value = 10
	}
	if data.Count.Unknown {
		data.Count.Value = 1
	}

	q, err := graphql.CreatePostgresCluster(context.Background(), *r.provider.client, data.Name.Value, data.Org.Value, data.Region.Value, data.Password.Value, data.Vmsize.Value, int(data.Volumesize.Value), int(data.Count.Value), "flyio/postgres")
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Postgres cluster", err.Error())
	}

	data = flyPgResourceData{
		Org:        types.String{Value: data.Org.Value},
		Name:       types.String{Value: data.Name.Value},
		Region:     types.String{Value: data.Name.Value},
		Username:   types.String{Value: q.CreatePostgresCluster.Username},
		Password:   types.String{Value: q.CreatePostgresCluster.Password},
		Vmsize:     types.String{Value: data.Vmsize.Value},
		Volumesize: types.Int64{Value: data.Volumesize.Value},
		Count:      types.Int64{Value: data.Count.Value},
	}

	tflog.Info(ctx, fmt.Sprintf("%+v", data))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r flyPgResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data flyPgResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	//var errList gqlerror.List
	//if errors.As(err, &errList) {
	//	for _, err := range errList {
	//		if err.Message == "Could not resolve " {
	//			return
	//		}
	//		resp.Diagnostics.AddError(err.Message, err.Path.String())
	//	}
	//} else if err != nil {
	//	resp.Diagnostics.AddError("Read: query failed", err.Error())
	//}
	//
	//data = flyPgResourceData{
	//}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r flyPgResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	resp.Diagnostics.AddError("The fly api does not allow updating certs once created", "Try deleting and then recreating the cert with new options")
	return
}

func (r flyPgResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data flyPgResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	resp.State.RemoveResource(ctx)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r flyPgResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

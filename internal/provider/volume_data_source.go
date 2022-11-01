package provider

import (
	"context"
	"github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = &volumeDataSource{}

// Matches getSchema
type volumeDataSourceOutput struct {
	Id         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Size       types.Int64  `tfsdk:"size"`
	Appid      types.String `tfsdk:"app"`
	Region     types.String `tfsdk:"region"`
	Internalid types.String `tfsdk:"internalid"`
}

func (v volumeDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "fly_volume"
}

func (v volumeDataSource) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
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
				MarkdownDescription: "Name of app to attach",
				Required:            true,
				Type:                types.StringType,
			},
			"size": {
				MarkdownDescription: "Size of volume in gb",
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
				Required:            true,
			},
		},
	}, nil
}

func NewVolumeDataSource() datasource.DataSource {
	return volumeDataSource{}
}

func (v volumeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data volumeDataSourceOutput

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	internalId := data.Internalid.Value
	app := data.Appid.Value

	query, err := graphql.VolumeQuery(context.Background(), v.gqlClient, app, internalId)
	if err != nil {
		resp.Diagnostics.AddError("Read: query failed", err.Error())
	}

	data = volumeDataSourceOutput{
		Id:         types.String{Value: query.App.Volume.Id},
		Name:       types.String{Value: query.App.Volume.Name},
		Size:       types.Int64{Value: int64(query.App.Volume.SizeGb)},
		Appid:      types.String{Value: data.Appid.Value},
		Region:     types.String{Value: query.App.Volume.Region},
		Internalid: types.String{Value: query.App.Volume.InternalId},
	}

	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

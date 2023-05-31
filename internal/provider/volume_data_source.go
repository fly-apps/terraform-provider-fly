package provider

import (
	"context"
	"regexp"

	basegql "github.com/Khan/genqlient/graphql"
	"github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &volumeDataSourceType{}
var _ datasource.DataSourceWithConfigure = &appDataSourceType{}

type volumeDataSourceType struct {
	client *basegql.Client
}

func NewVolumeDataSource() datasource.DataSource {
	return &volumeDataSourceType{}
}

func (d *volumeDataSourceType) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "fly_volume"
}

func (d *volumeDataSourceType) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config := req.ProviderData.(ProviderConfig)
	d.client = config.gqclient
}

// Matches Schema
type volumeDataSourceOutput struct {
	Id     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Size   types.Int64  `tfsdk:"size"`
	Appid  types.String `tfsdk:"app"`
	Region types.String `tfsdk:"region"`
}

func (d *volumeDataSourceType) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fly volume resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of volume",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^vol_[a-z0-9]+$`),
						"must start with \"vol_\"",
					),
				},
			},
			"app": schema.StringAttribute{
				MarkdownDescription: "Name of app attached to",
				Required:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "Size of volume in GB",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "name",
				Computed:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "region",
				Computed:            true,
			},
		},
	}
}

func (d *volumeDataSourceType) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data volumeDataSourceOutput

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	// strip leading vol_ off name
	internalId := data.Id.ValueString()[4:]
	app := data.Appid.ValueString()

	query, err := graphql.VolumeQuery(ctx, *d.client, app, internalId)
	if err != nil {
		resp.Diagnostics.AddError("Query failed", err.Error())
        return
	}

	// this query will currently still return success if it finds nothing, so check it:
	if query.App.Volume.Id == "" {
		resp.Diagnostics.AddError("Query failed", "Could not find matching volume")
	}

	data = volumeDataSourceOutput{
		Id:     types.StringValue(query.App.Volume.Id),
		Name:   types.StringValue(query.App.Volume.Name),
		Size:   types.Int64Value(int64(query.App.Volume.SizeGb)),
		Appid:  data.Appid,
		Region: types.StringValue(query.App.Volume.Region),
	}

	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

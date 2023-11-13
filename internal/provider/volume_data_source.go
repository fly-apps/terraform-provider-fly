package provider

import (
	"context"
	"github.com/fly-apps/terraform-provider-fly/pkg/apiv1"
	"regexp"

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
	config ProviderConfig
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
	d.config = req.ProviderData.(ProviderConfig)
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

	id := data.Id.ValueString()
	// New flaps based volumes don't have this prefix I'm pretty sure
	if id[:4] == "vol_" {
		// strip leading vol_ off name
		id = id[4:]
	}
	app := data.Appid.ValueString()

	machineAPI := apiv1.NewMachineAPI(d.config.httpClient, d.config.httpEndpoint)
	query, err := machineAPI.GetVolume(ctx, id, app)
	if err != nil {
		resp.Diagnostics.AddError("Query failed", err.Error())
		return
	}

	data = volumeDataSourceOutput{
		Id:     types.StringValue(query.ID),
		Name:   types.StringValue(query.Name),
		Size:   types.Int64Value(int64(query.SizeGb)),
		Appid:  types.StringValue(data.Appid.ValueString()),
		Region: types.StringValue(query.Region),
	}

	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

package provider

import (
	"context"
	"errors"
	"fmt"
	basegql "github.com/Khan/genqlient/graphql"
	"github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &ipDataSourceType{}
var _ datasource.DataSourceWithConfigure = &ipDataSourceType{}

func NewIpDataSource() datasource.DataSource {
	return &ipDataSourceType{}
}

type ipDataSourceType struct {
	client *basegql.Client
}

func (d *ipDataSourceType) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "fly_ip"
}

func (d *ipDataSourceType) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

    config := req.ProviderData.(ProviderConfig)
	d.client = config.gqclient
}

// Matches Schema
type ipDataSourceOutput struct {
	Id      types.String `tfsdk:"id"`
	Appid   types.String `tfsdk:"app"`
	Region  types.String `tfsdk:"region"`
	Address types.String `tfsdk:"address"`
	Type    types.String `tfsdk:"type"`
}

func (d *ipDataSourceType) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fly ip data source",
		Attributes: map[string]schema.Attribute{
			"address": schema.StringAttribute{
				MarkdownDescription: "IP address",
				Required:            true,
			},
			"app": schema.StringAttribute{
				MarkdownDescription: "Name of app attached to",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of address",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "v4 or v6",
				Computed:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "region",
				Computed:            true,
			},
		},
	}
}

func (d *ipDataSourceType) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ipDataSourceOutput

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	addr := data.Address.ValueString()
	app := data.Appid.ValueString()

	query, err := graphql.IpAddressQuery(context.Background(), *d.client, app, addr)
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

	region := query.App.IpAddress.Region
	if region == "" {
		region = "global"
	}

	data = ipDataSourceOutput{
		Id:      types.StringValue(query.App.IpAddress.Id),
		Appid:   data.Appid,
		Region:  types.StringValue(region),
		Type:    types.StringValue(string(query.App.IpAddress.Type)),
		Address: types.StringValue(query.App.IpAddress.Address),
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

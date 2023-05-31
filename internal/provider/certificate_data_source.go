package provider

import (
	"context"
	"errors"
	basegql "github.com/Khan/genqlient/graphql"
	"github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &certDataSourceType{}
var _ datasource.DataSourceWithConfigure = &certDataSourceType{}

type certDataSourceType struct {
	client *basegql.Client
}

func (d *certDataSourceType) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "fly_cert"
}

// Matches Schema
type certDataSourceOutput struct {
	Id                        types.String `tfsdk:"id"`
	Appid                     types.String `tfsdk:"app"`
	Dnsvalidationinstructions types.String `tfsdk:"dnsvalidationinstructions"`
	Dnsvalidationhostname     types.String `tfsdk:"dnsvalidationhostname"`
	Dnsvalidationtarget       types.String `tfsdk:"dnsvalidationtarget"`
	Hostname                  types.String `tfsdk:"hostname"`
	Check                     types.Bool   `tfsdk:"check"`
}

func (d *certDataSourceType) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fly certificate data source",
		Attributes: map[string]schema.Attribute{
			"app": schema.StringAttribute{
				MarkdownDescription: "Name of app attached to",
				Required:            true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "ID of certificate",
				Computed:            true,
			},
			"dnsvalidationinstructions": schema.StringAttribute{
				MarkdownDescription: "DnsValidationHostname",
				Computed:            true,
			},
			"dnsvalidationtarget": schema.StringAttribute{
				MarkdownDescription: "DnsValidationTarget",
				Computed:            true,
			},
			"dnsvalidationhostname": schema.StringAttribute{
				MarkdownDescription: "DnsValidationHostname",
				Computed:            true,
			},
			"check": schema.BoolAttribute{
				MarkdownDescription: "check",
				Computed:            true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "hostname",
				Required:            true,
			},
		},
	}
}

func NewCertDataSource() datasource.DataSource {
	return &certDataSourceType{}
}

func (d *certDataSourceType) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config := req.ProviderData.(ProviderConfig)
	d.client = config.gqclient
}

func (d *certDataSourceType) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data certDataSourceOutput

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	hostname := data.Hostname.ValueString()
	app := data.Appid.ValueString()

	query, err := graphql.GetCertificate(ctx, *d.client, app, hostname)
	var errList gqlerror.List
	if errors.As(err, &errList) {
		for _, err := range errList {
			if err.Message == "Could not resolve " {
				return
			}
			resp.Diagnostics.AddError(err.Message, err.Path.String())
		}
	} else if err != nil {
		resp.Diagnostics.AddError("Read: query failed", err.Error())
        return
	}

	data = certDataSourceOutput{
		Id:                        types.StringValue(query.App.Certificate.Id),
		Appid:                     data.Appid,
		Dnsvalidationinstructions: types.StringValue(query.App.Certificate.DnsValidationInstructions),
		Dnsvalidationhostname:     types.StringValue(query.App.Certificate.DnsValidationHostname),
		Dnsvalidationtarget:       types.StringValue(query.App.Certificate.DnsValidationTarget),
		Hostname:                  types.StringValue(query.App.Certificate.Hostname),
		Check:                     types.BoolValue(query.App.Certificate.Check),
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

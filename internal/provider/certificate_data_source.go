package provider

import (
	"context"
	"errors"
	"github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

var _ datasource.DataSourceWithConfigure = &certDataSource{}

// Matches getSchema
type certDataSourceOutput struct {
	Id                        types.String `tfsdk:"id"`
	Appid                     types.String `tfsdk:"app"`
	Dnsvalidationinstructions types.String `tfsdk:"dnsvalidationinstructions"`
	Dnsvalidationhostname     types.String `tfsdk:"dnsvalidationhostname"`
	Dnsvalidationtarget       types.String `tfsdk:"dnsvalidationtarget"`
	Hostname                  types.String `tfsdk:"hostname"`
	Check                     types.Bool   `tfsdk:"check"`
}

func (d certDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "fly_cert"
}

func (d certDataSource) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Fly certificate data source",
		Attributes: map[string]tfsdk.Attribute{
			"app": {
				MarkdownDescription: "Name of app that is attacjed",
				Required:            true,
				Type:                types.StringType,
			},
			"id": {
				MarkdownDescription: "ID of address",
				Computed:            true,
				Type:                types.StringType,
			},
			"dnsvalidationinstructions": {
				MarkdownDescription: "DnsValidationHostname",
				Type:                types.StringType,
				Computed:            true,
			},
			"dnsvalidationtarget": {
				MarkdownDescription: "DnsValidationTarget",
				Type:                types.StringType,
				Computed:            true,
			},
			"dnsvalidationhostname": {
				MarkdownDescription: "DnsValidationHostname",
				Type:                types.StringType,
				Computed:            true,
			},
			"check": {
				MarkdownDescription: "check",
				Type:                types.BoolType,
				Computed:            true,
			},
			"hostname": {
				MarkdownDescription: "hostname",
				Type:                types.StringType,
				Required:            true,
			},
		},
	}, nil
}

func newCertDataSource() datasource.DataSource {
	return &certDataSource{}
}

func (d certDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data certDataSourceOutput

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	hostname := data.Hostname.Value
	app := data.Appid.Value

	query, err := graphql.GetCertificate(context.Background(), d.gqlClient, app, hostname)
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
	}

	data = certDataSourceOutput{
		Id:                        types.String{Value: query.App.Certificate.Id},
		Appid:                     types.String{Value: data.Appid.Value},
		Dnsvalidationinstructions: types.String{Value: query.App.Certificate.DnsValidationInstructions},
		Dnsvalidationhostname:     types.String{Value: query.App.Certificate.DnsValidationHostname},
		Dnsvalidationtarget:       types.String{Value: query.App.Certificate.DnsValidationTarget},
		Hostname:                  types.String{Value: query.App.Certificate.Hostname},
		Check:                     types.Bool{Value: query.App.Certificate.Check},
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

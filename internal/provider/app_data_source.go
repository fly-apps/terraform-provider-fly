package provider

import (
	"context"
	"dov.dev/fly/fly-provider/graphql"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ tfsdk.DataSourceType = appDataSourceType{}
var _ tfsdk.DataSource = appDataSource{}

type appDataSourceType struct{}

// Matches getSchema
type appDataSourceOutput struct {
	Name           types.String `tfsdk:"name"`
	AppUrl         types.String `tfsdk:"appurl"`
	Hostname       types.String `tfsdk:"hostname"`
	Id             types.String `tfsdk:"id"`
	Status         types.String `tfsdk:"status"`
	Deployed       types.Bool   `tfsdk:"deployed"`
	Healthchecks   []string     `tfsdk:"healthchecks"`
	Ipaddresses    []string     `tfsdk:"ipaddresses"`
	Currentrelease types.String `tfsdk:"currentrelease"`
}

func (a appDataSourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Retrieve info about graphql app",

		Attributes: map[string]tfsdk.Attribute{
			"name": {
				MarkdownDescription: "Name of app",
				Type:                types.StringType,
				Required:            true,
			},
			"appurl": {
				Type:     types.StringType,
				Computed: true,
			},
			"hostname": {
				Type:     types.StringType,
				Computed: true,
			},
			"id": {
				Type:     types.StringType,
				Computed: true,
			},
			"status": {
				Type:     types.StringType,
				Computed: true,
			},
			"deployed": {
				Type:     types.BoolType,
				Computed: true,
			},
			"healthchecks": {
				Computed: true,
				Type:     types.ListType{ElemType: types.StringType},
			},
			"ipaddresses": {
				Computed: true,
				Type:     types.ListType{ElemType: types.StringType},
			},
			"currentrelease": {
				Type:     types.StringType,
				Computed: true,
			},
		},
	}, nil
}

func (a appDataSourceType) NewDataSource(ctx context.Context, in tfsdk.Provider) (tfsdk.DataSource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return appDataSource{
		provider: provider,
	}, diags
}

func (d appDataSource) Read(ctx context.Context, req tfsdk.ReadDataSourceRequest, resp *tfsdk.ReadDataSourceResponse) {
	var data appDataSourceOutput

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	appName := data.Name.Value

	queryresp, err := graphql.GetFullApp(context.Background(), *d.provider.client, appName)
	if err != nil {
		resp.Diagnostics.AddError("Query failed", err.Error())
	}

	a := appDataSourceOutput{
		Name:           types.String{Value: appName},
		AppUrl:         types.String{Value: string(queryresp.App.AppUrl)},
		Hostname:       types.String{Value: string(queryresp.App.Hostname)},
		Id:             types.String{Value: string(queryresp.App.Id)},
		Status:         types.String{Value: string(queryresp.App.Status)},
		Deployed:       types.Bool{Value: queryresp.App.Deployed},
		Currentrelease: types.String{Value: queryresp.App.CurrentRelease.Id},
		Healthchecks:   []string{},
		Ipaddresses:    []string{},
	}

	for _, s := range queryresp.App.HealthChecks.Nodes {
		a.Healthchecks = append(a.Healthchecks, s.Name+": "+s.Status)
	}

	for _, s := range queryresp.App.IpAddresses.Nodes {
		a.Ipaddresses = append(a.Ipaddresses, s.Address)
	}

	data = a

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

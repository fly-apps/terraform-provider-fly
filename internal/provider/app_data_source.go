package provider

import (
	"context"
	"github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSourceWithConfigure = &appDataSource{}

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
	//Secrets        types.Map    `tfsdk:"secrets"`
}

func (d appDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "fly_app"
}

func (appDataSource) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
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

func newAppDataSource() datasource.DataSource {
	return &appDataSource{}
}

func (d appDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data appDataSourceOutput

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	appName := data.Name.Value

	queryresp, err := graphql.GetFullApp(context.Background(), d.gqlClient, appName)
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

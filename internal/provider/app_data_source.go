package provider

import (
	"context"

	basegql "github.com/Khan/genqlient/graphql"
	"github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ datasource.DataSource = &appDataSourceType{}

func NewAppDataSource() datasource.DataSource {
    return &appDataSourceType{}
}

type appDataSourceType struct{
    client *basegql.Client
}

func (d *appDataSourceType) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = "fly_app"
}

func (d *appDataSourceType) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }

    d.client = req.ProviderData.(*basegql.Client)
    // Maybe wrapping the client in the tunneled client should be done here?
}
 
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

func (d *appDataSourceType) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieve info about graphql app",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of app",
				Required:            true,
			},
			"appurl": schema.StringAttribute{
				Computed: true,
			},
			"hostname": schema.StringAttribute{
				Computed: true,
			},
			"id": schema.StringAttribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"deployed": schema.BoolAttribute{
				Computed: true,
			},
			"healthchecks": schema.ListAttribute{
				Computed: true,
                ElementType: types.StringType,
			},
			"ipaddresses": schema.ListAttribute{
				Computed: true,
                ElementType: types.StringType,
			},
			"currentrelease": schema.StringAttribute{
				Computed: true,
			},
		},
    }
}

func (d *appDataSourceType) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data appDataSourceOutput

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	appName := data.Name.ValueString()

	queryresp, err := graphql.GetFullApp(context.Background(), *d.client, appName)
	if err != nil {
		resp.Diagnostics.AddError("Query failed", err.Error())
	}

	a := appDataSourceOutput{
		Name:           types.StringValue(appName),
		AppUrl:         types.StringValue(queryresp.App.AppUrl),
		Hostname:       types.StringValue(queryresp.App.Hostname),
		Id:             types.StringValue(queryresp.App.Id),
		Status:         types.StringValue(queryresp.App.Status),
		Deployed:       types.BoolValue(queryresp.App.Deployed),
		Currentrelease: types.StringValue(queryresp.App.CurrentRelease.Id),
		// Healthchecks:   []string{},
		// Ipaddresses:    []string{},
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

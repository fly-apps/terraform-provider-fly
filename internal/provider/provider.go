package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Khan/genqlient/graphql"
	providerGraphql "github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/fly-apps/terraform-provider-fly/internal/utils"
	"github.com/fly-apps/terraform-provider-fly/internal/wg"
	hreq "github.com/imroc/req/v3"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &flyProvider{}

type flyProvider struct {
	configured   bool
	version      string
	token        string
	httpEndpoint string
	client       *graphql.Client
	httpClient   *hreq.Client
}

type flyProviderData struct {
	FlyToken             types.String `tfsdk:"fly_api_token"`
	FlyHttpEndpoint      types.String `tfsdk:"fly_http_endpoint"`
	UseInternalTunnel    types.Bool   `tfsdk:"useinternaltunnel"`
	InternalTunnelOrg    types.String `tfsdk:"internaltunnelorg"`
	InternalTunnelRegion types.String `tfsdk:"internaltunnelregion"`
}

func (p *flyProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
    resp.TypeName = "fly"
}

func (p *flyProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data flyProviderData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var token string
	if data.FlyToken.IsUnknown() {
		resp.Diagnostics.AddWarning(
			"Unable to create client",
			"Cannot use unknown value as token",
		)
		return
	}
	if data.FlyToken.IsNull() {
		token = os.Getenv("FLY_API_TOKEN")
	} else {
		token = data.FlyToken.ValueString()
	}
	if token == "" {
		resp.Diagnostics.AddError(
			"Unable to find token",
			"token cannot be an empty string",
		)
		return
	}

	p.token = token

	endpoint, exists := os.LookupEnv("FLY_HTTP_ENDPOINT")
	httpEndpoint := "127.0.0.1:4280"
	if !data.FlyHttpEndpoint.IsNull() && !data.FlyHttpEndpoint.IsUnknown() {
		httpEndpoint = data.FlyHttpEndpoint.ValueString()
	} else if exists {
		httpEndpoint = endpoint
	}

	p.httpEndpoint = httpEndpoint

	enableTracing := false
	_, ok := os.LookupEnv("DEBUG")
	if ok {
		enableTracing = true
		resp.Diagnostics.AddWarning("Debug mode enabled", "Debug mode enabled, this will add the Fly-Force-Trace header to all graphql requests")
	}

	p.httpClient = hreq.C()

	if enableTracing {
		p.httpClient.SetCommonHeader("Fly-Force-Trace", "true")
		p.httpClient = hreq.C().DevMode()
	}

	p.httpClient.SetCommonHeader("Authorization", "Bearer "+p.token)
	p.httpClient.SetTimeout(2 * time.Minute)

	// TODO: Make timeout configurable
	h := http.Client{Timeout: 60 * time.Second, Transport: &utils.Transport{UnderlyingTransport: http.DefaultTransport, Token: token, Ctx: ctx, EnableDebugTrace: enableTracing}}
	client := graphql.NewClient("https://api.fly.io/graphql", &h)
	p.client = &client

	if data.UseInternalTunnel.ValueBool() {
		org, err := providerGraphql.Organization(context.Background(), client, data.InternalTunnelOrg.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Could not resolve organization", err.Error())
			return
		}
		tunnel, err := wg.Establish(ctx, org.Organization.Id, data.InternalTunnelRegion.ValueString(), token, &client)
		if err != nil {
			resp.Diagnostics.AddError("failed to open internal tunnel", err.Error())
			return
		}
		p.httpClient.SetDial(tunnel.NetStack().DialContext)
		p.httpEndpoint = "_api.internal:4280"
	}
	p.configured = true
    resp.DataSourceData = p.client
    resp.ResourceData = p.client
}

func (p *flyProvider) Resources(ctx context.Context) []func() resource.Resource {
    return []func() resource.Resource {
// 		"fly_app":     flyAppResourceType{},
// 		"fly_volume":  flyVolumeResourceType{},
// 		"fly_ip":      flyIpResourceType{},
// 		"fly_cert":    flyCertResourceType{},
// 		"fly_machine": flyMachineResourceType{},
// 	}
    }
}

func (p *flyProvider) DataSources(_ context.Context) []func() datasource.DataSource {
    return []func() datasource.DataSource {
        NewAppDataSource, // fly_app
		// "fly_cert":   certDataSourceType{},
		// "fly_ip":     ipDataSourceType{},
		// "fly_volume": volumeDataSourceType{},
	}
}

func (p *flyProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
    resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"fly_api_token": schema.StringAttribute{
				MarkdownDescription: "fly.io api token. If not set checks env for FLY_API_TOKEN",
				Optional:            true,
			},
			"fly_http_endpoint": schema.StringAttribute{
				MarkdownDescription: "Where the provider should look to find the fly http endpoint",
				Optional:            true,
			},
			"useinternaltunnel": schema.BoolAttribute{
				Optional: true,
			},
			"internaltunnelorg": schema.StringAttribute{
				Optional: true,
			},
			"internaltunnelregion": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &flyProvider{
			version: version,
		}
	}
}

// convertProviderType is a helper function for NewResource and NewDataSource
// implementations to associate the concrete provider type. Alternatively,
// this helper can be skipped and the provider type can be directly type
// asserted (e.g. provider: in.(*provider)), however using this can prevent
// potential panics.
func convertProviderType(in provider.Provider) (flyProvider, diag.Diagnostics) {
	var diags diag.Diagnostics

	p, ok := in.(*flyProvider)

	if !ok {
		diags.AddError(
			"Unexpected Provider Instance Type",
			fmt.Sprintf("While creating the data source or resource, an unexpected provider type (%T) was received. This is always a bug in the provider code and should be reported to the provider developers.", p),
		)
		return flyProvider{}, diags
	}

	if p == nil {
		diags.AddError(
			"Unexpected Provider Instance Type",
			"While creating the data source or resource, an unexpected empty provider instance was received. This is always a bug in the provider code and should be reported to the provider developers.",
		)
		return flyProvider{}, diags
	}

	return *p, diags
}

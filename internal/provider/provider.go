package provider

import (
	"context"
	"fmt"
	"github.com/Khan/genqlient/graphql"
	"github.com/fly-apps/terraform-provider-fly/internal/utils"
	hreq "github.com/imroc/req/v3"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tfsdkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ tfsdkprovider.Provider = &provider{}

type provider struct {
	configured   bool
	version      string
	token        string
	httpEndpoint string
	client       *graphql.Client
	httpClient   *hreq.Client
}

type providerData struct {
	FlyToken        types.String `tfsdk:"fly_api_token"`
	FlyHttpEndpoint types.String `tfsdk:"fly_http_endpoint"`
}

func (p *provider) Configure(ctx context.Context, req tfsdkprovider.ConfigureRequest, resp *tfsdkprovider.ConfigureResponse) {
	var data providerData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var token string
	if data.FlyToken.Unknown {
		resp.Diagnostics.AddWarning(
			"Unable to create client",
			"Cannot use unknown value as token",
		)
		return
	}
	if data.FlyToken.Null || data.FlyToken.Unknown {
		token = os.Getenv("FLY_API_TOKEN")
	} else {
		token = data.FlyToken.Value
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
	if !data.FlyHttpEndpoint.Null && !data.FlyHttpEndpoint.Unknown {
		httpEndpoint = data.FlyHttpEndpoint.Value
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

//	hclient := hreq.C().DevMode()
	hclient := hreq.C()
	p.httpClient = hclient

	p.httpClient.SetCommonHeader("Authorization", "Bearer "+p.token)
	p.httpClient.SetTimeout(2 * time.Minute)

	if enableTracing {
		p.httpClient.SetCommonHeader("Fly-Force-Trace", "true")
	}

	// TODO: Make timeout configurable
	h := http.Client{Timeout: 60 * time.Second, Transport: &utils.Transport{UnderlyingTransport: http.DefaultTransport, Token: token, Ctx: ctx, EnableDebugTrace: enableTracing}}
	client := graphql.NewClient("https://api.fly.io/graphql", &h)
	p.client = &client

	p.configured = true
}

func (p *provider) GetResources(ctx context.Context) (map[string]tfsdkprovider.ResourceType, diag.Diagnostics) {

	return map[string]tfsdkprovider.ResourceType{
		"fly_app":     flyAppResourceType{},
		"fly_volume":  flyVolumeResourceType{},
		"fly_ip":      flyIpResourceType{},
		"fly_cert":    flyCertResourceType{},
		"fly_machine": flyMachineResourceType{},
	}, nil
}

func (p *provider) GetDataSources(ctx context.Context) (map[string]tfsdkprovider.DataSourceType, diag.Diagnostics) {
	return map[string]tfsdkprovider.DataSourceType{
		"fly_app":  appDataSourceType{},
		"fly_cert": certDataSourceType{},
		"fly_ip":   ipDataSourceType{},
	}, nil
}

func (p *provider) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"fly_api_token": {
				MarkdownDescription: "fly.io api token. If not set checks env for FLY_API_TOKEN",
				Optional:            true,
				Type:                types.StringType,
			},
			"fly_http_endpoint": {
				MarkdownDescription: "Where the provider should look to find the fly http endpoint",
				Optional:            true,
				Type:                types.StringType,
			},
		},
	}, nil
}

func New(version string) func() tfsdkprovider.Provider {
	return func() tfsdkprovider.Provider {
		return &provider{
			version: version,
		}
	}
}

// convertProviderType is a helper function for NewResource and NewDataSource
// implementations to associate the concrete provider type. Alternatively,
// this helper can be skipped and the provider type can be directly type
// asserted (e.g. provider: in.(*provider)), however using this can prevent
// potential panics.
func convertProviderType(in tfsdkprovider.Provider) (provider, diag.Diagnostics) {
	var diags diag.Diagnostics

	p, ok := in.(*provider)

	if !ok {
		diags.AddError(
			"Unexpected Provider Instance Type",
			fmt.Sprintf("While creating the data source or resource, an unexpected provider type (%T) was received. This is always a bug in the provider code and should be reported to the provider developers.", p),
		)
		return provider{}, diags
	}

	if p == nil {
		diags.AddError(
			"Unexpected Provider Instance Type",
			"While creating the data source or resource, an unexpected empty provider instance was received. This is always a bug in the provider code and should be reported to the provider developers.",
		)
		return provider{}, diags
	}

	return *p, diags
}

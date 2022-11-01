package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type flyDataSource struct {
	providerClients
}

func (d *flyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.configure(req.ProviderData, &resp.Diagnostics)
}

type flyResource struct {
	providerClients
}

func (r *flyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.configure(req.ProviderData, &resp.Diagnostics)
}

type appDataSource struct {
	flyDataSource
}
type certDataSource struct {
	flyDataSource
}
type ipDataSource struct {
	flyDataSource
}
type volumeDataSource struct {
	flyDataSource
}

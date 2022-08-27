package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/fly-apps/terraform-provider-fly/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

var _ tfsdk.ResourceType = flyAppResourceType{}
var _ tfsdk.Resource = flyAppResource{}
var _ tfsdk.ResourceWithImportState = flyAppResource{}

type flyAppResourceType struct{}

type flyAppResourceData struct {
	Name   types.String `tfsdk:"name"`
	Org    types.String `tfsdk:"org"`
	OrgId  types.String `tfsdk:"orgid"`
	AppUrl types.String `tfsdk:"appurl"`
}

func (ar flyAppResourceType) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Fly app resource",

		Attributes: map[string]tfsdk.Attribute{
			"name": {
				MarkdownDescription: "Name of application",
				Required:            true,
				Type:                types.StringType,
			},
			"org": {
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "Optional org slug to operate upon",
				Type:                types.StringType,
			},
			"orgid": {
				Computed:            true,
				MarkdownDescription: "readonly orgid",
				Type:                types.StringType,
			},
			"appurl": {
				Computed:            true,
				MarkdownDescription: "readonly appUrl",
				Type:                types.StringType,
			},
		},
	}, nil
}

func (ar flyAppResourceType) NewResource(_ context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return flyAppResource{
		provider: provider,
	}, diags
}

type flyAppResource struct {
	provider provider
}

func (r flyAppResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data flyAppResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Org.Unknown {
		defaultOrg, err := utils.GetDefaultOrg(*r.provider.client)
		if err != nil {
			resp.Diagnostics.AddError("Could not detect default organization", err.Error())
			return
		}
		data.OrgId.Value = defaultOrg.Id
		data.Org.Value = defaultOrg.Name
	} else {
		org, err := graphql.Organization(context.Background(), *r.provider.client, data.Org.Value)
		if err != nil {
			resp.Diagnostics.AddError("Could not resolve organization", err.Error())
			return
		}
		data.OrgId.Value = org.Organization.Id
	}
	mresp, err := graphql.CreateAppMutation(context.Background(), *r.provider.client, data.Name.Value, data.OrgId.Value)
	if err != nil {
		resp.Diagnostics.AddError("Create app failed", err.Error())
		return
	}

	data = flyAppResourceData{
		Org:    types.String{Value: mresp.CreateApp.App.Organization.Slug},
		OrgId:  types.String{Value: mresp.CreateApp.App.Organization.Id},
		Name:   types.String{Value: mresp.CreateApp.App.Name},
		AppUrl: types.String{Value: mresp.CreateApp.App.AppUrl},
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r flyAppResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data flyAppResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	query, err := graphql.GetFullApp(context.Background(), *r.provider.client, data.Name.Value)
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

	data = flyAppResourceData{
		Name:   types.String{Value: query.App.Name},
		Org:    types.String{Value: query.App.Organization.Slug},
		OrgId:  types.String{Value: query.App.Organization.Id},
		AppUrl: types.String{Value: query.App.AppUrl},
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r flyAppResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var plan flyAppResourceData

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var state flyAppResourceData
	diags = resp.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, fmt.Sprintf("existing: %+v, new: %+v", state, plan))

	if !plan.Org.Unknown && plan.Org.Value != state.Org.Value {
		resp.Diagnostics.AddError("Can't mutate org of existing app", "Can't switch org"+state.Org.Value+" to "+plan.Org.Value)
	}
	if !plan.Name.Null && plan.Name.Value != state.Name.Value {
		resp.Diagnostics.AddError("Can't mutate Name of existing app", "Can't switch name "+state.Name.Value+" to "+plan.Name.Value)
	}

	resp.State.Set(ctx, state)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r flyAppResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data flyAppResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	_, err := graphql.DeleteAppMutation(context.Background(), *r.provider.client, data.Name.Value)
	var errList gqlerror.List
	if errors.As(err, &errList) {
		for _, err := range errList {
			resp.Diagnostics.AddError(err.Message, err.Path.String())
		}
	} else if err != nil {
		resp.Diagnostics.AddError("Delete app failed", err.Error())
	}

	resp.State.RemoveResource(ctx)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r flyAppResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("name"), req, resp)
}

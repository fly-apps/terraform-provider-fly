package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/fly-apps/terraform-provider-fly/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfsdkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

var _ tfsdkprovider.ResourceType = flyAppResourceType{}
var _ resource.Resource = flyAppResource{}
var _ resource.ResourceWithImportState = flyAppResource{}

type flyAppResourceType struct{}

type flyAppResourceData struct {
	Name   types.String `tfsdk:"name"`
	Org    types.String `tfsdk:"org"`
	OrgId  types.String `tfsdk:"orgid"`
	AppUrl types.String `tfsdk:"appurl"`
	Id     types.String `tfsdk:"id"`
	//Secrets types.Map    `tfsdk:"secrets"`
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
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.RequiresReplace(),
				},
			},
			"org": {
				Computed:            true,
				Optional:            true,
				MarkdownDescription: "Optional org slug to operate upon",
				Type:                types.StringType,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.RequiresReplace(),
				},
			},
			"orgid": {
				Computed:            true,
				MarkdownDescription: "readonly orgid",
				Type:                types.StringType,
			},
			"id": {
				Computed:            true,
				MarkdownDescription: "readonly app id",
				Type:                types.StringType,
			},
			"appurl": {
				Computed:            true,
				MarkdownDescription: "readonly appUrl",
				Type:                types.StringType,
			},
			//"secrets": {
			//	Sensitive:           true,
			//	Optional:            true,
			//	MarkdownDescription: "App secrets",
			//	Type:                types.MapType{ElemType: types.StringType},
			//},
		},
	}, nil
}

func (ar flyAppResourceType) NewResource(_ context.Context, in tfsdkprovider.Provider) (resource.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return flyAppResource{
		provider: provider,
	}, diags
}

type flyAppResource struct {
	provider provider
}

func (r flyAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
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
		Id:     types.String{Value: mresp.CreateApp.App.Id},
	}

	//if len(data.Secrets.Elems) > 0 {
	//	var rawSecrets map[string]string
	//	data.Secrets.ElementsAs(context.Background(), &rawSecrets, false)
	//	var secrets []graphql.SecretInput
	//	for k, v := range rawSecrets {
	//		secrets = append(secrets, graphql.SecretInput{
	//			Key:   k,
	//			Value: v,
	//		})
	//	}
	//	_, err := graphql.SetSecrets(context.Background(), *r.provider.client, graphql.SetSecretsInput{
	//		AppId:      data.Id.Value,
	//		Secrets:    secrets,
	//		ReplaceAll: true,
	//	})
	//	if err != nil {
	//		resp.Diagnostics.AddError("Could not set rawSecrets", err.Error())
	//		return
	//	}
	//	data.Secrets = utils.KVToTfMap(rawSecrets, types.StringType)
	//}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r flyAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state flyAppResourceData

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	query, err := graphql.GetFullApp(context.Background(), *r.provider.client, state.Name.Value)
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

	data := flyAppResourceData{
		Name:   types.String{Value: query.App.Name},
		Org:    types.String{Value: query.App.Organization.Slug},
		OrgId:  types.String{Value: query.App.Organization.Id},
		AppUrl: types.String{Value: query.App.AppUrl},
		Id:     types.String{Value: query.App.Id},
	}

	//if !state.Secrets.Null && !state.Secrets.Unknown {
	//	data.Secrets = state.Secrets
	//}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r flyAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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

	//if len(plan.Secrets.Elems) > 0 {
	//	var rawSecrets map[string]string
	//	plan.Secrets.ElementsAs(context.Background(), &rawSecrets, false)
	//	var secrets []graphql.SecretInput
	//	for k, v := range rawSecrets {
	//		secrets = append(secrets, graphql.SecretInput{
	//			Key:   k,
	//			Value: v,
	//		})
	//	}
	//	_, err := graphql.SetSecrets(context.Background(), *r.provider.client, graphql.SetSecretsInput{
	//		AppId:      state.Id.Value,
	//		Secrets:    secrets,
	//		ReplaceAll: true,
	//	})
	//	if err != nil {
	//		resp.Diagnostics.AddError("Could not set rawSecrets", err.Error())
	//		return
	//	}
	//	state.Secrets = utils.KVToTfMap(rawSecrets, types.StringType)
	//}

	resp.State.Set(ctx, state)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r flyAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
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

func (r flyAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

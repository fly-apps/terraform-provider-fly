package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/fly-apps/terraform-provider-fly/graphql"
	"github.com/fly-apps/terraform-provider-fly/internal/provider/modifiers"
	"github.com/fly-apps/terraform-provider-fly/internal/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"strings"
	"time"
)

var _ resource.ResourceWithConfigure = &flyAppResource{}
var _ resource.ResourceWithImportState = &flyAppResource{}

type appSecret struct {
	Value     types.String `tfsdk:"value"`
	Digest    types.String `tfsdk:"digest"`
	CreatedAt types.String `tfsdk:"created_at"`
}

var secretType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"value":      types.StringType,
		"digest":     types.StringType,
		"created_at": types.StringType,
	},
}

type secretMap map[string]appSecret

//var secretMapType = types.MapType{ElemType: secretType}

//type secretMap types.Map

type flyAppResourceData struct {
	Name    types.String `tfsdk:"name"`
	Org     types.String `tfsdk:"org"`
	OrgId   types.String `tfsdk:"orgid"`
	AppUrl  types.String `tfsdk:"appurl"`
	Id      types.String `tfsdk:"id"`
	Secrets secretMap    `tfsdk:"secrets"`
}

func (d *flyAppResourceData) setSecret(name string, secret appSecret) {
	if d.Secrets == nil {
		d.Secrets = make(secretMap)
	}
	d.Secrets[name] = secret
}

func (d *flyAppResourceData) secretValues() map[string]string {
	values := make(map[string]string, len(d.Secrets))
	for k, v := range d.Secrets {
		values[k] = v.Value.Value
	}
	return values
}

func (d *flyAppResourceData) updateFromApi(a graphql.AppFragment) {
	d.Name = types.String{Value: a.Name}
	d.Org = types.String{Value: a.Organization.Slug}
	d.OrgId = types.String{Value: a.Organization.Id}
	d.AppUrl = types.String{Value: a.AppUrl}
	d.Id = types.String{Value: a.Id}
}

func (d *flyAppResourceData) updateSecretsFromApi(a graphql.AppFragment) {
	for k := range d.Secrets {
		d.updateSecretFromApi(k, a)
	}
}

func (d *flyAppResourceData) updateSecretFromApi(name string, a graphql.AppFragment) {
	s := d.Secrets[name]
	for _, as := range a.Secrets {
		if as.Name != name {
			continue
		}
		if as.Digest != s.Digest.Value || as.CreatedAt.Format(time.RFC3339) != s.CreatedAt.Value {
			d.Secrets[name] = appSecret{
				Digest:    types.String{Value: as.Digest},
				CreatedAt: types.String{Value: as.CreatedAt.Format(time.RFC3339)},
				Value:     types.String{Unknown: true},
			}
		}
		return
	}
	// Not found in app, so secret was removed outside of Terraform
	delete(d.Secrets, name)
}

func (r flyAppResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "fly_app"
}

func (r flyAppResource) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
	var secretValueUnchanged modifiers.UseStateForUnknownIfFunc = func(ctx context.Context, req tfsdk.ModifyAttributePlanRequest) (ok bool, diags diag.Diagnostics) {
		valuePath := req.AttributePath.ParentPath().AtName("value")
		var stateValue, configValue types.String
		diags.Append(req.State.GetAttribute(ctx, valuePath, &stateValue)...)
		diags.Append(req.Config.GetAttribute(ctx, valuePath, &configValue)...)
		return stateValue.Equal(configValue), diags
	}

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
			"secrets": {
				Optional:    true,
				Description: "Secret environment variables. Keys are case sensitive and are used as environment variable names. Does not override existing secrets added outside of Terraform.",
				Attributes: tfsdk.MapNestedAttributes(map[string]tfsdk.Attribute{
					"value": {
						Type:      types.StringType,
						Sensitive: true,
						Required:  true,
					},
					"digest": {
						Type:          types.StringType,
						Computed:      true,
						PlanModifiers: []tfsdk.AttributePlanModifier{modifiers.UseStateForUnknownIf(secretValueUnchanged)},
					},
					"created_at": {
						Type:          types.StringType,
						Computed:      true,
						PlanModifiers: []tfsdk.AttributePlanModifier{modifiers.UseStateForUnknownIf(secretValueUnchanged)},
					},
				}),
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
		},
	}, nil
}

func newAppResource() resource.Resource {
	return &flyAppResource{}
}

type flyAppResource struct {
	flyResource
}

func (r flyAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data flyAppResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Org.Unknown {
		defaultOrg, err := utils.GetDefaultOrg(r.gqlClient)
		if err != nil {
			resp.Diagnostics.AddError("Could not detect default organization", err.Error())
			return
		}
		data.OrgId.Value = defaultOrg.Id
		data.Org.Value = defaultOrg.Name
	} else {
		org, err := graphql.Organization(context.Background(), r.gqlClient, data.Org.Value)
		if err != nil {
			resp.Diagnostics.AddError("Could not resolve organization", err.Error())
			return
		}
		data.OrgId.Value = org.Organization.Id
	}
	mresp, err := graphql.CreateAppMutation(context.Background(), r.gqlClient, data.Name.Value, data.OrgId.Value)
	if err != nil {
		resp.Diagnostics.AddError("Create app failed", err.Error())
		return
	}
	data.updateFromApi(mresp.CreateApp.App)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(data.Secrets) > 0 {
		r.setSecrets(ctx, data.Name.Value, data.Secrets, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r flyAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state flyAppResourceData

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	query, err := graphql.GetApp(context.Background(), r.gqlClient, state.Name.Value)
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

	state.updateFromApi(query.App)
	state.updateSecretsFromApi(query.App)

	diags = resp.State.Set(ctx, &state)
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

	// Unset secrets that were removed from config
	var removedSecrets []string
	for k := range state.Secrets {
		if _, ok := plan.Secrets[k]; !ok {
			removedSecrets = append(removedSecrets, k)
			delete(state.Secrets, k)
		}
	}
	if len(removedSecrets) > 0 {
		_, err := graphql.UnsetSecrets(ctx, r.gqlClient, state.Name.Value, removedSecrets)
		if err != nil {
			resp.Diagnostics.AddError("UnsetSecrets failed", err.Error())
		} else {
			resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
		}
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Set secrets that were changed or newly appeared in config
	newSecrets := make(secretMap)
	for k, ps := range plan.Secrets {
		if s, ok := state.Secrets[k]; !ok || !s.Value.Equal(ps.Value) {
			newSecrets[k] = ps
		}
	}
	if len(newSecrets) > 0 {
		r.setSecrets(ctx, state.Name.Value, newSecrets, &resp.Diagnostics)
		for k, s := range newSecrets {
			state.setSecret(k, s)
		}
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r flyAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data flyAppResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	_, err := graphql.DeleteAppMutation(context.Background(), r.gqlClient, data.Name.Value)
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

// setSecrets sets the app's secrets specified in the secret map `secrets` and
// writes back `digest` and `createdAt` attributes.
func (r flyAppResource) setSecrets(ctx context.Context, appName string, secrets secretMap, diags *diag.Diagnostics) {
	inputs := make([]graphql.SecretInput, len(secrets))
	i := 0
	for k, v := range secrets {
		inputs[i] = graphql.SecretInput{
			Key:   k,
			Value: v.Value.Value,
		}
		i++
	}

	resp, err := graphql.SetSecrets(ctx, r.gqlClient, graphql.SetSecretsInput{
		AppId:      appName,
		Secrets:    inputs,
		ReplaceAll: false,
	})
	if err != nil {
		var errList gqlerror.List
		if errors.As(err, &errList) && len(errList) == 1 && strings.Contains(errList[0].Message, "No change detected") {
			diags.AddWarning("SetSecrets was no-op", err.Error())
			// Secrets may have been added outside of Terraform. To prevent unknown value errors, we need to fetch
			// digest and createdAt attributes for secrets that are missing those values in state
			diags.AddWarning("State may have drifted", "Secrets may have been added or changed outside of Terraform. Refreshing secret state, you may need to re-apply your Terraform config.")
			r.getSecretComputedValues(ctx, appName, secrets, diags)
		} else {
			diags.AddError("SetSecrets errored", err.Error())
		}
		return
	}

	for _, s := range resp.SetSecrets.App.Secrets {
		if _, ok := secrets[s.Name]; !ok {
			continue
		}
		secrets[s.Name] = appSecret{
			Value:     secrets[s.Name].Value,
			Digest:    types.String{Value: s.Digest},
			CreatedAt: types.String{Value: s.CreatedAt.Format(time.RFC3339)},
		}
	}
}

func (r flyAppResource) getSecretComputedValues(ctx context.Context, appName string, secrets secretMap, diags *diag.Diagnostics) {
	apiSecrets, err := graphql.GetSecrets(ctx, r.gqlClient, appName)
	if err != nil {
		diags.AddError("GetSecrets failed", err.Error())
	}
	for _, as := range apiSecrets.App.Secrets {
		if s, ok := secrets[as.Name]; ok {
			secrets[as.Name] = appSecret{
				Value:     s.Value,
				Digest:    types.String{Value: as.Digest},
				CreatedAt: types.String{Value: as.CreatedAt.Format(time.RFC3339)},
			}
		}
	}
}

func (r flyAppResource) unsetSecrets(ctx context.Context, appName string, secretKeys []string, diags *diag.Diagnostics) {
	_, err := graphql.UnsetSecrets(ctx, r.gqlClient, appName, secretKeys)
	if err != nil {
		diags.AddError("UnsetSecrets failed", err.Error())
	}
}

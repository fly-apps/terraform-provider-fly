package provider

import (
	"context"
	"dov.dev/fly/fly-provider/graphql"
	"errors"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

var _ tfsdk.ResourceType = flyCertResourceType{}
var _ tfsdk.Resource = flyCertResource{}
var _ tfsdk.ResourceWithImportState = flyCertResource{}

type flyCertResourceType struct{}

type flyCertResource struct {
	provider provider
}

type flyCertResourceData struct {
	Id                        types.String `tfsdk:"id"`
	Appid                     types.String `tfsdk:"app"`
	Dnsvalidationinstructions types.String `tfsdk:"dnsvalidationinstructions"`
	Dnsvalidationhostname     types.String `tfsdk:"dnsvalidationhostname"`
	Dnsvalidationtarget       types.String `tfsdk:"dnsvalidationtarget"`
	Hostname                  types.String `tfsdk:"hostname"`
	Check                     types.Bool   `tfsdk:"check"`
}

func (t flyCertResourceType) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		MarkdownDescription: "Fly certificate resource",
		Attributes: map[string]tfsdk.Attribute{
			"app": {
				MarkdownDescription: "Name of app to attach",
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

func (t flyCertResourceType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	provider, diags := convertProviderType(in)

	return flyCertResource{
		provider: provider,
	}, diags
}

func (cr flyCertResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data flyCertResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	q, err := graphql.AddCertificate(context.Background(), *cr.provider.client, data.Appid.Value, data.Hostname.Value)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create cert", err.Error())
	}

	data = flyCertResourceData{
		Id:                        types.String{Value: q.AddCertificate.Certificate.Id},
		Appid:                     types.String{Value: data.Appid.Value},
		Dnsvalidationinstructions: types.String{Value: q.AddCertificate.Certificate.DnsValidationInstructions},
		Dnsvalidationhostname:     types.String{Value: q.AddCertificate.Certificate.DnsValidationHostname},
		Dnsvalidationtarget:       types.String{Value: q.AddCertificate.Certificate.DnsValidationTarget},
		Hostname:                  types.String{Value: q.AddCertificate.Certificate.Hostname},
		Check:                     types.Bool{Value: q.AddCertificate.Certificate.Check},
	}

	tflog.Info(ctx, fmt.Sprintf("%+v", data))

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (cr flyCertResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data flyCertResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	hostname := data.Hostname.Value
	app := data.Appid.Value

	query, err := graphql.GetCertificate(context.Background(), *cr.provider.client, app, hostname)
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

	data = flyCertResourceData{
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
	if resp.Diagnostics.HasError() {
		return
	}
}

func (cr flyCertResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	resp.Diagnostics.AddError("The fly api does not allow updating certs once created", "Try deleting and then recreating the cert with new options")
	return
}

func (cr flyCertResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data flyCertResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	_, err := graphql.DeleteCertificate(context.Background(), *cr.provider.client, data.Appid.Value, data.Hostname.Value)
	if err != nil {
		resp.Diagnostics.AddError("Delete cert failed", err.Error())
	}

	resp.State.RemoveResource(ctx)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (cr flyCertResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

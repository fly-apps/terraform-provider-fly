package modifiers

import (
	"context"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

type UseStateForUnknownIfFunc func(ctx context.Context, req tfsdk.ModifyAttributePlanRequest) (bool, diag.Diagnostics)

// UseStateForUnknownIf is like UseStateForUnknown, but conditional
func UseStateForUnknownIf(condition UseStateForUnknownIfFunc) tfsdk.AttributePlanModifier {
	return useStateForUnknownIfModifier{condition}
}

// useStateForUnknownIfModifier implements the UseStateForUnknownIf
// AttributePlanModifier.
type useStateForUnknownIfModifier struct {
	condition UseStateForUnknownIfFunc
}

// Modify copies the attribute's prior state to the attribute plan if the prior
// state value is not null.
func (r useStateForUnknownIfModifier) Modify(ctx context.Context, req tfsdk.ModifyAttributePlanRequest, resp *tfsdk.ModifyAttributePlanResponse) {
	if req.AttributeState == nil || resp.AttributePlan == nil || req.AttributeConfig == nil {
		return
	}

	// if we have no state value, there's nothing to preserve
	if req.AttributeState.IsNull() {
		return
	}

	// if it's not planned to be the unknown value, stick with the concrete plan
	if !resp.AttributePlan.IsUnknown() {
		return
	}

	// if the config is the unknown value, use the unknown value otherwise, interpolation gets messed up
	if req.AttributeConfig.IsUnknown() {
		return
	}

	ok, diags := r.condition(ctx, req)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	if !ok {
		return
	}

	resp.AttributePlan = req.AttributeState
}

func (r useStateForUnknownIfModifier) Description(context.Context) string {
	return "Once set, the value of this attribute in state will not change as long as the given condition holds."
}

func (r useStateForUnknownIfModifier) MarkdownDescription(ctx context.Context) string {
	return r.Description(ctx)
}

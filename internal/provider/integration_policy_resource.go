package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dkurasov/terraform-provider-incidentgarden/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type integrationPolicyResource struct{ provider *providerData }

type integrationPolicyModel struct {
	ID             types.String    `tfsdk:"id"`
	Organization   types.String    `tfsdk:"organization"`
	TeamID         types.String    `tfsdk:"team_id"`
	IntegrationID  types.String    `tfsdk:"integration_id"`
	AppliesToAll   types.Bool      `tfsdk:"applies_to_all"`
	Name           types.String    `tfsdk:"name"`
	Description    types.String    `tfsdk:"description"`
	Position       types.Int64     `tfsdk:"position"`
	IsActive       types.Bool      `tfsdk:"is_active"`
	Conditions     types.Object    `tfsdk:"conditions"`
	Action         types.Object    `tfsdk:"action"`
	ConditionsJSON jsonStringValue `tfsdk:"conditions_json"`
	ActionJSON     jsonStringValue `tfsdk:"action_json"`
	CreatedAt      types.String    `tfsdk:"created_at"`
	UpdatedAt      types.String    `tfsdk:"updated_at"`
}

func NewIntegrationPolicyResource() resource.Resource { return &integrationPolicyResource{} }

func (r *integrationPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_policy"
}

func (r *integrationPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{Description: "A conditional integration policy exposed by /policies. Scope fields are immutable.", Attributes: mergeAttributes(identityAttributes(), map[string]schema.Attribute{
		"team_id": schema.StringAttribute{Required: true, PlanModifiers: replace}, "integration_id": schema.StringAttribute{Optional: true, PlanModifiers: replace, Description: "Required when applies_to_all is false; omitted when true."}, "applies_to_all": schema.BoolAttribute{Required: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}}, "name": schema.StringAttribute{Required: true}, "description": schema.StringAttribute{Optional: true}, "position": schema.Int64Attribute{Required: true}, "is_active": schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}},
		"conditions": policyConditionsAttribute(), "action": policyActionAttribute(),
		"conditions_json": schema.StringAttribute{CustomType: jsonStringType{}, Optional: true, Computed: true, Validators: []validator.String{validJSONValidator{}}, DeprecationMessage: "Use conditions instead.", Description: "Deprecated PolicyConditions compatibility field."},
		"action_json":     schema.StringAttribute{CustomType: jsonStringType{}, Optional: true, Computed: true, Validators: []validator.String{validJSONValidator{}}, DeprecationMessage: "Use action instead.", Description: "Deprecated policy action compatibility field."},
	})}
}

func (r *integrationPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResource(req, resp, &r.provider)
}

func (r *integrationPolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config integrationPolicyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if configured(config.Conditions) && configured(config.ConditionsJSON) {
		resp.Diagnostics.AddAttributeError(path.Root("conditions"), "Conflicting conditions fields", "Configure conditions or the deprecated conditions_json, not both.")
	}
	if configured(config.Action) && configured(config.ActionJSON) {
		resp.Diagnostics.AddAttributeError(path.Root("action"), "Conflicting action fields", "Configure action or the deprecated action_json, not both.")
		return
	}
	if !configured(config.Action) && !configured(config.ActionJSON) {
		resp.Diagnostics.AddAttributeError(path.Root("action"), "Missing policy action", "Configure one typed action block.")
		return
	}
	if !configured(config.Action) {
		return
	}
	values := config.Action.Attributes()
	actionType := values["type"].(types.String)
	if actionType.IsUnknown() {
		return
	}
	value := values["value"].(types.Object)
	require := func(names ...string) bool {
		if !configured(value) {
			return false
		}
		expected := make(map[string]struct{}, len(names))
		for _, name := range names {
			expected[name] = struct{}{}
			if !configured(value.Attributes()[name]) {
				return false
			}
		}
		for name, attribute := range value.Attributes() {
			if _, ok := expected[name]; !ok && configured(attribute) {
				return false
			}
		}
		return true
	}
	valid := false
	switch actionType.ValueString() {
	case "drop", "suppress":
		valid = !configured(value)
	case "override_severity":
		valid = require("value")
	case "add_labels":
		valid = require("labels")
	case "hold_outside_window":
		valid = require("start", "end", "days_of_week")
	}
	if !valid {
		resp.Diagnostics.AddAttributeError(path.Root("action"), "Invalid policy action", "The action value does not match its type: drop/suppress have no value; override_severity requires value.value; add_labels requires value.labels; hold_outside_window requires value.start, value.end, and value.days_of_week.")
	}
}

func policyBody(p integrationPolicyModel, create, useTypedConditions, useTypedAction bool) (map[string]any, error) {
	var actionValue any
	switch {
	case useTypedAction && !p.Action.IsNull() && !p.Action.IsUnknown():
		actionValue = objectValue(p.Action)
	case !p.ActionJSON.IsNull() && !p.ActionJSON.IsUnknown():
		action, e := rawJSON(p.ActionJSON.StringValue, "{}")
		if e != nil {
			return nil, e
		}
		if e = json.Unmarshal(action, &actionValue); e != nil {
			return nil, e
		}
	case !p.Action.IsNull() && !p.Action.IsUnknown():
		actionValue = objectValue(p.Action)
	default:
		return nil, fmt.Errorf("action must be configured")
	}
	body := map[string]any{"name": p.Name.ValueString(), "description": stringPointer(p.Description), "position": p.Position.ValueInt64(), "actions": []any{actionValue}}
	switch {
	case useTypedConditions && !p.Conditions.IsNull() && !p.Conditions.IsUnknown():
		body["conditions"] = objectValue(p.Conditions)
	case !p.ConditionsJSON.IsNull() && !p.ConditionsJSON.IsUnknown():
		raw, e := rawJSON(p.ConditionsJSON.StringValue, "{}")
		if e != nil {
			return nil, e
		}
		var v any
		if e = json.Unmarshal(raw, &v); e != nil {
			return nil, e
		}
		body["conditions"] = v
	case !p.Conditions.IsNull() && !p.Conditions.IsUnknown():
		body["conditions"] = objectValue(p.Conditions)
	}
	if create {
		body["team_id"] = p.TeamID.ValueString()
		body["applies_to_all"] = p.AppliesToAll.ValueBool()
		body["integration_id"] = stringPointer(p.IntegrationID)
	} else if !p.IsActive.IsNull() && !p.IsActive.IsUnknown() {
		body["is_active"] = p.IsActive.ValueBool()
	}
	return body, nil
}

func (r *integrationPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p integrationPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	org, e := organization(p.Organization, r.provider.DefaultOrganization)
	if e != nil {
		resp.Diagnostics.AddError("Missing organization", e.Error())
		return
	}
	var configuredConditions, configuredAction types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("conditions"), &configuredConditions)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("action"), &configuredAction)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, e := policyBody(p, true, configured(configuredConditions), configured(configuredAction))
	if e != nil {
		resp.Diagnostics.AddError("Invalid policy JSON", e.Error())
		return
	}
	v, e := r.provider.Client.CreateIntegrationPolicy(ctx, org, body)
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "create integration policy", e)
		return
	}
	if deactivateAfterCreate(p.IsActive) {
		created := v
		v, e = r.provider.Client.UpdateIntegrationPolicy(ctx, org, v.ID, map[string]any{"is_active": false})
		if e != nil {
			p.apply(org, created)
			resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
			apiDiagnostic(&resp.Diagnostics, "deactivate integration policy after create", e)
			return
		}
	}
	p.apply(org, v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}

func (r *integrationPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s integrationPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, e := r.provider.Client.GetIntegrationPolicy(ctx, s.Organization.ValueString(), s.ID.ValueString())
	if client.IsNotFound(e) {
		resp.State.RemoveResource(ctx)
		return
	}
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "read integration policy", e)
		return
	}
	s.apply(s.Organization.ValueString(), v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}

func (r *integrationPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p integrationPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var configuredConditions, configuredAction types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("conditions"), &configuredConditions)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("action"), &configuredAction)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, e := policyBody(p, false, configured(configuredConditions), configured(configuredAction))
	if e != nil {
		resp.Diagnostics.AddError("Invalid policy JSON", e.Error())
		return
	}
	v, e := r.provider.Client.UpdateIntegrationPolicy(ctx, p.Organization.ValueString(), p.ID.ValueString(), body)
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "update integration policy", e)
		return
	}
	p.apply(p.Organization.ValueString(), v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}

func (r *integrationPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s integrationPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	e := r.provider.Client.DeleteIntegrationPolicy(ctx, s.Organization.ValueString(), s.ID.ValueString())
	if client.IsNotFound(e) {
		return
	}
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "delete integration policy", e)
	}
}

func (r *integrationPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}

func (m *integrationPolicyModel) apply(org string, v client.IntegrationPolicy) {
	m.ID = types.StringValue(v.ID)
	m.Organization = types.StringValue(org)
	m.TeamID = types.StringValue(v.TeamID)
	m.IntegrationID = nullableString(v.IntegrationID)
	m.AppliesToAll = types.BoolValue(v.AppliesToAll)
	m.Name = types.StringValue(v.Name)
	m.Description = nullableString(v.Description)
	m.Position = types.Int64Value(v.Position)
	m.IsActive = types.BoolValue(v.IsActive)
	m.ConditionsJSON, _ = canonicalJSONString(v.Conditions, "{}")
	m.Conditions, _ = objectFromJSON(v.Conditions, policyConditionsTypes)
	var actions []json.RawMessage
	if json.Unmarshal(v.Actions, &actions) == nil && len(actions) == 1 {
		m.ActionJSON, _ = canonicalJSONString(actions[0], "{}")
		m.Action, _ = objectFromJSON(actions[0], policyActionTypes)
	} else {
		m.ActionJSON = jsonStringValue{StringValue: types.StringValue("{}")}
		m.Action = types.ObjectNull(policyActionTypes)
	}
	m.CreatedAt = types.StringValue(v.CreatedAt)
	m.UpdatedAt = types.StringValue(v.UpdatedAt)
}

var _ resource.ResourceWithImportState = (*integrationPolicyResource)(nil)
var _ resource.ResourceWithValidateConfig = (*integrationPolicyResource)(nil)

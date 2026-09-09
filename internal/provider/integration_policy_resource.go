package provider

import (
	"context"
	"encoding/json"
	"github.com/dkurasov/incidentgarden-terraform-provider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type integrationPolicyResource struct{ provider *providerData }
type integrationPolicyModel struct {
	ID             types.String `tfsdk:"id"`
	Organization   types.String `tfsdk:"organization"`
	TeamID         types.String `tfsdk:"team_id"`
	IntegrationID  types.String `tfsdk:"integration_id"`
	AppliesToAll   types.Bool   `tfsdk:"applies_to_all"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Position       types.Int64  `tfsdk:"position"`
	IsActive       types.Bool   `tfsdk:"is_active"`
	ConditionsJSON types.String `tfsdk:"conditions_json"`
	ActionJSON     types.String `tfsdk:"action_json"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func NewIntegrationPolicyResource() resource.Resource { return &integrationPolicyResource{} }
func (r *integrationPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_policy"
}
func (r *integrationPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{Description: "A conditional integration policy exposed by /policies. Scope fields are immutable.", Attributes: mergeAttributes(identityAttributes(), map[string]schema.Attribute{
		"team_id": schema.StringAttribute{Required: true, PlanModifiers: replace}, "integration_id": schema.StringAttribute{Optional: true, PlanModifiers: replace, Description: "Required when applies_to_all is false; omitted when true."}, "applies_to_all": schema.BoolAttribute{Required: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}}, "name": schema.StringAttribute{Required: true}, "description": schema.StringAttribute{Optional: true}, "position": schema.Int64Attribute{Required: true}, "is_active": schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}, "conditions_json": schema.StringAttribute{Optional: true, Computed: true, Description: "JSON PolicyConditions object."}, "action_json": schema.StringAttribute{Required: true, Description: "JSON for the policy's single discriminated action object."},
	})}
}
func (r *integrationPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResource(req, resp, &r.provider)
}
func policyBody(p integrationPolicyModel, create bool) (map[string]any, error) {
	action, e := rawJSON(p.ActionJSON, "{}")
	if e != nil {
		return nil, e
	}
	var actionValue any
	if e = json.Unmarshal(action, &actionValue); e != nil {
		return nil, e
	}
	body := map[string]any{"name": p.Name.ValueString(), "description": stringPointer(p.Description), "position": p.Position.ValueInt64(), "actions": []any{actionValue}}
	if !p.ConditionsJSON.IsNull() && !p.ConditionsJSON.IsUnknown() {
		raw, e := rawJSON(p.ConditionsJSON, "{}")
		if e != nil {
			return nil, e
		}
		var v any
		json.Unmarshal(raw, &v)
		body["conditions"] = v
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
	body, e := policyBody(p, true)
	if e != nil {
		resp.Diagnostics.AddError("Invalid policy JSON", e.Error())
		return
	}
	v, e := r.provider.Client.CreateIntegrationPolicy(ctx, org, body)
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "create integration policy", e)
		return
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
	body, e := policyBody(p, false)
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
	m.ConditionsJSON, _ = canonicalJSON(v.Conditions, "{}")
	var actions []json.RawMessage
	if json.Unmarshal(v.Actions, &actions) == nil && len(actions) == 1 {
		m.ActionJSON, _ = canonicalJSON(actions[0], "{}")
	} else {
		m.ActionJSON = types.StringValue("{}")
	}
	m.CreatedAt = types.StringValue(v.CreatedAt)
	m.UpdatedAt = types.StringValue(v.UpdatedAt)
}

var _ resource.ResourceWithImportState = (*integrationPolicyResource)(nil)

package provider

import (
	"context"

	"github.com/dkurasov/terraform-provider-incidentgarden/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type escalationPolicyResource struct{ provider *providerData }
type escalationPolicyModel struct {
	ID           types.String          `tfsdk:"id"`
	Organization types.String          `tfsdk:"organization"`
	Name         types.String          `tfsdk:"name"`
	Description  types.String          `tfsdk:"description"`
	TeamID       types.String          `tfsdk:"team_id"`
	Steps        []escalationStepModel `tfsdk:"step"`
	CreatedAt    types.String          `tfsdk:"created_at"`
	UpdatedAt    types.String          `tfsdk:"updated_at"`
}
type escalationStepModel struct {
	ID                      types.String `tfsdk:"id"`
	Position                types.Int64  `tfsdk:"position"`
	TargetType              types.String `tfsdk:"target_type"`
	ScheduleID              types.String `tfsdk:"schedule_id"`
	UserID                  types.String `tfsdk:"user_id"`
	TeamID                  types.String `tfsdk:"team_id"`
	ActivationOffsetMinutes types.Int64  `tfsdk:"activation_offset_minutes"`
}

func NewEscalationPolicyResource() resource.Resource { return &escalationPolicyResource{} }
func (r *escalationPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_escalation_policy"
}
func (r *escalationPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "An ordered Incident Garden escalation policy.", Attributes: mergeAttributes(identityAttributes(), map[string]schema.Attribute{
		"name": schema.StringAttribute{Required: true}, "description": schema.StringAttribute{Optional: true}, "team_id": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"step": schema.ListNestedAttribute{Required: true, Description: "Ordered steps. Configure in ascending activation_offset_minutes because the API sorts by that value.", NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true}, "position": schema.Int64Attribute{Computed: true}, "target_type": schema.StringAttribute{Required: true}, "schedule_id": schema.StringAttribute{Optional: true}, "user_id": schema.StringAttribute{Optional: true}, "team_id": schema.StringAttribute{Optional: true}, "activation_offset_minutes": schema.Int64Attribute{Required: true},
		}}},
	})}
}
func (r *escalationPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResource(req, resp, &r.provider)
}
func expandSteps(values []escalationStepModel) []client.EscalationStep {
	out := make([]client.EscalationStep, len(values))
	for i, v := range values {
		out[i] = client.EscalationStep{TargetType: v.TargetType.ValueString(), ScheduleID: stringPointer(v.ScheduleID), UserID: stringPointer(v.UserID), TeamID: stringPointer(v.TeamID), ActivationOffsetMinutes: v.ActivationOffsetMinutes.ValueInt64()}
	}
	return out
}
func flattenSteps(values []client.EscalationStep) []escalationStepModel {
	out := make([]escalationStepModel, len(values))
	for i, v := range values {
		out[i] = escalationStepModel{ID: types.StringValue(v.ID), Position: types.Int64Value(v.Position), TargetType: types.StringValue(v.TargetType), ScheduleID: nullableString(v.ScheduleID), UserID: nullableString(v.UserID), TeamID: nullableString(v.TeamID), ActivationOffsetMinutes: types.Int64Value(v.ActivationOffsetMinutes)}
	}
	return out
}
func (r *escalationPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p escalationPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	org, e := organization(p.Organization, r.provider.DefaultOrganization)
	if e != nil {
		resp.Diagnostics.AddError("Missing organization", e.Error())
		return
	}
	body := map[string]any{"name": p.Name.ValueString(), "team_id": p.TeamID.ValueString(), "steps": expandSteps(p.Steps)}
	if description := stringPointer(p.Description); description != nil {
		body["description"] = *description
	}
	v, e := r.provider.Client.CreateEscalationPolicy(ctx, org, body)
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "create escalation policy", e)
		return
	}
	p.apply(org, v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *escalationPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s escalationPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, e := r.provider.Client.GetEscalationPolicy(ctx, s.Organization.ValueString(), s.ID.ValueString())
	if client.IsNotFound(e) {
		resp.State.RemoveResource(ctx)
		return
	}
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "read escalation policy", e)
		return
	}
	s.apply(s.Organization.ValueString(), v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *escalationPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p escalationPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body := map[string]any{"name": p.Name.ValueString(), "steps": expandSteps(p.Steps)}
	if description := stringPointer(p.Description); description != nil {
		body["description"] = *description
	}
	v, e := r.provider.Client.UpdateEscalationPolicy(ctx, p.Organization.ValueString(), p.ID.ValueString(), body)
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "update escalation policy", e)
		return
	}
	p.apply(p.Organization.ValueString(), v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *escalationPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s escalationPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	e := r.provider.Client.DeleteEscalationPolicy(ctx, s.Organization.ValueString(), s.ID.ValueString())
	if client.IsNotFound(e) {
		return
	}
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "delete escalation policy; repoint integrations and account for retained alerts first", e)
	}
}
func (r *escalationPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}
func (m *escalationPolicyModel) apply(org string, v client.EscalationPolicy) {
	m.ID = types.StringValue(v.ID)
	m.Organization = types.StringValue(org)
	m.Name = types.StringValue(v.Name)
	m.Description = nullableString(v.Description)
	m.TeamID = types.StringValue(v.TeamID)
	m.Steps = flattenSteps(v.Steps)
	m.CreatedAt = types.StringValue(v.CreatedAt)
	m.UpdatedAt = types.StringValue(v.UpdatedAt)
}

var _ resource.ResourceWithImportState = (*escalationPolicyResource)(nil)

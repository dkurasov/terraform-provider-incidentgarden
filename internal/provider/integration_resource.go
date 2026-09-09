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

type integrationResource struct{ provider *providerData }
type integrationModel struct {
	ID                  types.String `tfsdk:"id"`
	Organization        types.String `tfsdk:"organization"`
	Name                types.String `tfsdk:"name"`
	Type                types.String `tfsdk:"type"`
	TeamID              types.String `tfsdk:"team_id"`
	EscalationPolicyID  types.String `tfsdk:"escalation_policy_id"`
	SeverityMappingJSON types.String `tfsdk:"severity_mapping_json"`
	SettingsJSON        types.String `tfsdk:"settings_json"`
	IsActive            types.Bool   `tfsdk:"is_active"`
	APIKeyPrefix        types.String `tfsdk:"api_key_prefix"`
	APIKey              types.String `tfsdk:"api_key"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

func NewIntegrationResource() resource.Resource { return &integrationResource{} }
func (r *integrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}
func (r *integrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "An alert ingestion integration. api_key is returned only at creation and is preserved in state.", Attributes: mergeAttributes(identityAttributes(), map[string]schema.Attribute{
		"name": schema.StringAttribute{Required: true}, "type": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}, "team_id": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}, "escalation_policy_id": schema.StringAttribute{Required: true},
		"severity_mapping_json": schema.StringAttribute{Optional: true, Computed: true, Description: "JSON SeverityMapping object."}, "settings_json": schema.StringAttribute{Optional: true, Computed: true, Description: "Type-specific JSON settings object or null."}, "is_active": schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}},
		"api_key_prefix": schema.StringAttribute{Computed: true}, "api_key": schema.StringAttribute{Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
	})}
}
func (r *integrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResource(req, resp, &r.provider)
}
func integrationBody(p integrationModel, create bool) (map[string]any, error) {
	body := map[string]any{"name": p.Name.ValueString(), "escalation_policy_id": p.EscalationPolicyID.ValueString()}
	if create {
		body["type"] = p.Type.ValueString()
		body["team_id"] = p.TeamID.ValueString()
	}
	if !p.SeverityMappingJSON.IsNull() && !p.SeverityMappingJSON.IsUnknown() {
		raw, e := rawJSON(p.SeverityMappingJSON, "{}")
		if e != nil {
			return nil, e
		}
		var v any
		json.Unmarshal(raw, &v)
		body["severity_mapping"] = v
	}
	if !p.SettingsJSON.IsNull() && !p.SettingsJSON.IsUnknown() {
		raw, e := rawJSON(p.SettingsJSON, "null")
		if e != nil {
			return nil, e
		}
		var v any
		json.Unmarshal(raw, &v)
		body["settings"] = v
	}
	if !create && !p.IsActive.IsNull() && !p.IsActive.IsUnknown() {
		body["is_active"] = p.IsActive.ValueBool()
	}
	return body, nil
}
func (r *integrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p integrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	org, e := organization(p.Organization, r.provider.DefaultOrganization)
	if e != nil {
		resp.Diagnostics.AddError("Missing organization", e.Error())
		return
	}
	body, e := integrationBody(p, true)
	if e != nil {
		resp.Diagnostics.AddError("Invalid integration JSON", e.Error())
		return
	}
	v, e := r.provider.Client.CreateIntegration(ctx, org, body)
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "create integration", e)
		return
	}
	p.apply(org, v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *integrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s integrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, e := r.provider.Client.GetIntegration(ctx, s.Organization.ValueString(), s.ID.ValueString())
	if client.IsNotFound(e) {
		resp.State.RemoveResource(ctx)
		return
	}
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "read integration", e)
		return
	}
	s.apply(s.Organization.ValueString(), v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *integrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p integrationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, e := integrationBody(p, false)
	if e != nil {
		resp.Diagnostics.AddError("Invalid integration JSON", e.Error())
		return
	}
	v, e := r.provider.Client.UpdateIntegration(ctx, p.Organization.ValueString(), p.ID.ValueString(), body)
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "update integration", e)
		return
	}
	p.apply(p.Organization.ValueString(), v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *integrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s integrationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	e := r.provider.Client.DeleteIntegration(ctx, s.Organization.ValueString(), s.ID.ValueString())
	if client.IsNotFound(e) {
		return
	}
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "delete integration; resolve or remove open alerts first", e)
	}
}
func (r *integrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}
func (m *integrationModel) apply(org string, v client.Integration) {
	m.ID = types.StringValue(v.ID)
	m.Organization = types.StringValue(org)
	m.Name = types.StringValue(v.Name)
	m.Type = types.StringValue(v.Type)
	m.TeamID = types.StringValue(v.TeamID)
	m.EscalationPolicyID = types.StringValue(v.EscalationPolicyID)
	raw, _ := json.Marshal(v.SeverityMapping)
	m.SeverityMappingJSON, _ = canonicalJSON(raw, "{}")
	m.SettingsJSON, _ = canonicalJSON(v.Settings, "null")
	m.IsActive = types.BoolValue(v.IsActive)
	m.APIKeyPrefix = types.StringValue(v.APIKeyPrefix)
	if v.APIKey != "" {
		m.APIKey = types.StringValue(v.APIKey)
	}
	m.CreatedAt = types.StringValue(v.CreatedAt)
	m.UpdatedAt = types.StringValue(v.UpdatedAt)
}

var _ resource.ResourceWithImportState = (*integrationResource)(nil)

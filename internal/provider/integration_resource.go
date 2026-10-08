package provider

import (
	"context"
	"encoding/json"
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

type integrationResource struct{ provider *providerData }

type integrationModel struct {
	ID                  types.String    `tfsdk:"id"`
	Organization        types.String    `tfsdk:"organization"`
	Name                types.String    `tfsdk:"name"`
	Type                types.String    `tfsdk:"type"`
	TeamID              types.String    `tfsdk:"team_id"`
	EscalationPolicyID  types.String    `tfsdk:"escalation_policy_id"`
	SeverityMappingJSON jsonStringValue `tfsdk:"severity_mapping_json"`
	SeverityMapping     types.Object    `tfsdk:"severity_mapping"`
	SettingsJSON        jsonStringValue `tfsdk:"settings_json"`
	IsActive            types.Bool      `tfsdk:"is_active"`
	APIKeyPrefix        types.String    `tfsdk:"api_key_prefix"`
	APIKey              types.String    `tfsdk:"api_key"`
	CreatedAt           types.String    `tfsdk:"created_at"`
	UpdatedAt           types.String    `tfsdk:"updated_at"`
}

func NewIntegrationResource() resource.Resource { return &integrationResource{} }

func (r *integrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

func (r *integrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "An alert ingestion integration. api_key is returned only at creation and is preserved in state.", Attributes: mergeAttributes(identityAttributes(), map[string]schema.Attribute{
		"name": schema.StringAttribute{Required: true}, "type": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}, "team_id": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}}, "escalation_policy_id": schema.StringAttribute{Required: true},
		"severity_mapping":      severityMappingAttribute(),
		"severity_mapping_json": schema.StringAttribute{CustomType: jsonStringType{}, Optional: true, Computed: true, Validators: []validator.String{validJSONValidator{}}, DeprecationMessage: "Use severity_mapping instead.", Description: "Deprecated JSON SeverityMapping compatibility field."},
		"settings_json":         schema.StringAttribute{CustomType: jsonStringType{}, Optional: true, Computed: true, Validators: []validator.String{validJSONValidator{}}, Description: "Type-specific JSON settings object or null; JSON formatting and object key order are ignored."}, "is_active": schema.BoolAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}},
		"api_key_prefix": schema.StringAttribute{Computed: true}, "api_key": schema.StringAttribute{Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
	})}
}

func (r *integrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResource(req, resp, &r.provider)
}

func (r *integrationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config integrationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if configured(config.SeverityMapping) && configured(config.SeverityMappingJSON) {
		resp.Diagnostics.AddAttributeError(path.Root("severity_mapping"), "Conflicting severity mapping fields", "Configure severity_mapping or the deprecated severity_mapping_json, not both.")
	}
}

func integrationBody(p integrationModel, create, useTypedSeverity bool) (map[string]any, error) {
	body := map[string]any{"name": p.Name.ValueString(), "escalation_policy_id": p.EscalationPolicyID.ValueString()}
	if create {
		body["type"] = p.Type.ValueString()
		body["team_id"] = p.TeamID.ValueString()
	}
	switch {
	case useTypedSeverity && !p.SeverityMapping.IsNull() && !p.SeverityMapping.IsUnknown():
		body["severity_mapping"] = objectValue(p.SeverityMapping)
	case !p.SeverityMappingJSON.IsNull() && !p.SeverityMappingJSON.IsUnknown():
		raw, e := rawJSON(p.SeverityMappingJSON.StringValue, "{}")
		if e != nil {
			return nil, e
		}
		var v any
		if e = json.Unmarshal(raw, &v); e != nil {
			return nil, e
		}
		body["severity_mapping"] = v
	case !p.SeverityMapping.IsNull() && !p.SeverityMapping.IsUnknown():
		body["severity_mapping"] = objectValue(p.SeverityMapping)
	}
	if !p.SettingsJSON.IsNull() && !p.SettingsJSON.IsUnknown() {
		raw, e := rawJSON(p.SettingsJSON.StringValue, "null")
		if e != nil {
			return nil, e
		}
		var v any
		if e = json.Unmarshal(raw, &v); e != nil {
			return nil, e
		}
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
	var configuredSeverity types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("severity_mapping"), &configuredSeverity)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, e := integrationBody(p, true, configured(configuredSeverity))
	if e != nil {
		resp.Diagnostics.AddError("Invalid integration JSON", e.Error())
		return
	}
	v, e := r.provider.Client.CreateIntegration(ctx, org, body)
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "create integration", e)
		return
	}
	if deactivateAfterCreate(p.IsActive) {
		created := v
		v, e = r.provider.Client.UpdateIntegration(ctx, org, v.ID, map[string]any{"is_active": false})
		if e != nil {
			p.apply(org, created)
			resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
			apiDiagnostic(&resp.Diagnostics, "deactivate integration after create", e)
			return
		}
		preserveCreateOnlyIntegrationFields(created, &v)
	}
	p.apply(org, v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}

func preserveCreateOnlyIntegrationFields(created client.Integration, updated *client.Integration) {
	if updated.APIKey == "" {
		updated.APIKey = created.APIKey
	}
	if updated.APIKeyPrefix == "" {
		updated.APIKeyPrefix = created.APIKeyPrefix
	}
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
	var configuredSeverity types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("severity_mapping"), &configuredSeverity)...)
	if resp.Diagnostics.HasError() {
		return
	}
	body, e := integrationBody(p, false, configured(configuredSeverity))
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
	m.SeverityMappingJSON, _ = canonicalJSONString(v.SeverityMapping, "{}")
	m.SeverityMapping, _ = objectFromJSON(v.SeverityMapping, severityMappingTypes)
	m.SettingsJSON, _ = canonicalJSONString(v.Settings, "null")
	m.IsActive = types.BoolValue(v.IsActive)
	m.APIKeyPrefix = types.StringValue(v.APIKeyPrefix)
	if v.APIKey != "" {
		m.APIKey = types.StringValue(v.APIKey)
	}
	m.CreatedAt = types.StringValue(v.CreatedAt)
	m.UpdatedAt = types.StringValue(v.UpdatedAt)
}

var _ resource.ResourceWithImportState = (*integrationResource)(nil)
var _ resource.ResourceWithValidateConfig = (*integrationResource)(nil)

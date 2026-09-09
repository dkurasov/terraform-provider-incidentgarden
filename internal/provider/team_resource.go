package provider

import (
	"context"

	"github.com/dkurasov/terraform-provider-incidentgarden/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type teamResource struct{ provider *providerData }
type teamModel struct {
	ID           types.String `tfsdk:"id"`
	Organization types.String `tfsdk:"organization"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func NewTeamResource() resource.Resource { return &teamResource{} }
func (r *teamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}
func (r *teamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "An Incident Garden team.", Attributes: mergeAttributes(identityAttributes(), map[string]schema.Attribute{
		"name": schema.StringAttribute{Required: true}, "description": schema.StringAttribute{Optional: true},
	})}
}
func (r *teamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResource(req, resp, &r.provider)
}
func (r *teamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan teamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	org, err := organization(plan.Organization, r.provider.DefaultOrganization)
	if err != nil {
		resp.Diagnostics.AddError("Missing organization", err.Error())
		return
	}
	body := map[string]any{"name": plan.Name.ValueString()}
	if description := stringPointer(plan.Description); description != nil {
		body["description"] = *description
	}
	remote, err := r.provider.Client.CreateTeam(ctx, org, body)
	if err != nil {
		apiDiagnostic(&resp.Diagnostics, "create team", err)
		return
	}
	plan.apply(org, remote)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *teamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := r.provider.Client.GetTeam(ctx, state.Organization.ValueString(), state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		apiDiagnostic(&resp.Diagnostics, "read team", err)
		return
	}
	state.apply(state.Organization.ValueString(), remote)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
func (r *teamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan teamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := r.provider.Client.UpdateTeam(ctx, plan.Organization.ValueString(), plan.ID.ValueString(), map[string]any{"name": plan.Name.ValueString(), "description": stringPointer(plan.Description)})
	if err != nil {
		apiDiagnostic(&resp.Diagnostics, "update team", err)
		return
	}
	plan.apply(plan.Organization.ValueString(), remote)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
func (r *teamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state teamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.provider.Client.DeleteTeam(ctx, state.Organization.ValueString(), state.ID.ValueString())
	if client.IsNotFound(err) {
		return
	}
	if err != nil {
		apiDiagnostic(&resp.Diagnostics, "delete team; remove dependent schedules, escalation policies, integrations, and escalation steps first", err)
	}
}
func (r *teamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}
func (m *teamModel) apply(org string, v client.Team) {
	m.ID = types.StringValue(v.ID)
	m.Organization = types.StringValue(org)
	m.Name = types.StringValue(v.Name)
	m.Description = nullableString(v.Description)
	m.CreatedAt = types.StringValue(v.CreatedAt)
	m.UpdatedAt = types.StringValue(v.UpdatedAt)
}

var _ resource.ResourceWithImportState = (*teamResource)(nil)

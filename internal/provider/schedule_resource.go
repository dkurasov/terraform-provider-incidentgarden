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

type scheduleResource struct{ provider *providerData }

type scheduleModel struct {
	ID           types.String `tfsdk:"id"`
	Organization types.String `tfsdk:"organization"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	TeamID       types.String `tfsdk:"team_id"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func NewScheduleResource() resource.Resource { return &scheduleResource{} }

func (r *scheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_schedule"
}

func (r *scheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Description: "An on-call schedule. Rotations and overrides have independent API lifecycles and are intentionally outside this MVP resource.", Attributes: mergeAttributes(identityAttributes(), map[string]schema.Attribute{
		"name": schema.StringAttribute{Required: true}, "description": schema.StringAttribute{Optional: true}, "team_id": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
	})}
}

func (r *scheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResource(req, resp, &r.provider)
}

func (r *scheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p scheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	org, e := organization(p.Organization, r.provider.DefaultOrganization)
	if e != nil {
		resp.Diagnostics.AddError("Missing organization", e.Error())
		return
	}
	body := map[string]any{"name": p.Name.ValueString(), "team_id": p.TeamID.ValueString()}
	if description := stringPointer(p.Description); description != nil {
		body["description"] = *description
	}
	v, e := r.provider.Client.CreateSchedule(ctx, org, body)
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "create schedule", e)
		return
	}
	p.apply(org, v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}

func (r *scheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s scheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, e := r.provider.Client.GetSchedule(ctx, s.Organization.ValueString(), s.ID.ValueString())
	if client.IsNotFound(e) {
		resp.State.RemoveResource(ctx)
		return
	}
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "read schedule", e)
		return
	}
	s.apply(s.Organization.ValueString(), v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}

func (r *scheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p scheduleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, e := r.provider.Client.UpdateSchedule(ctx, p.Organization.ValueString(), p.ID.ValueString(), map[string]any{"name": p.Name.ValueString(), "description": stringPointer(p.Description)})
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "update schedule", e)
		return
	}
	p.apply(p.Organization.ValueString(), v)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}

func (r *scheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s scheduleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	e := r.provider.Client.DeleteSchedule(ctx, s.Organization.ValueString(), s.ID.ValueString())
	if client.IsNotFound(e) {
		return
	}
	if e != nil {
		apiDiagnostic(&resp.Diagnostics, "delete schedule; remove escalation steps that reference it first", e)
	}
}

func (r *scheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importCompositeID(ctx, req, resp)
}

func (m *scheduleModel) apply(org string, v client.Schedule) {
	m.ID = types.StringValue(v.ID)
	m.Organization = types.StringValue(org)
	m.Name = types.StringValue(v.Name)
	m.Description = nullableString(v.Description)
	m.TeamID = types.StringValue(v.TeamID)
	m.CreatedAt = types.StringValue(v.CreatedAt)
	m.UpdatedAt = types.StringValue(v.UpdatedAt)
}

var _ resource.ResourceWithImportState = (*scheduleResource)(nil)

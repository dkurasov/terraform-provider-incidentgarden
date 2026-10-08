package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/dkurasov/terraform-provider-incidentgarden/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type teamMembersResource struct{ provider *providerData }

type teamMembersModel struct {
	ID           types.String      `tfsdk:"id"`
	Organization types.String      `tfsdk:"organization"`
	TeamID       types.String      `tfsdk:"team_id"`
	Members      []teamMemberModel `tfsdk:"member"`
}

type teamMemberModel struct {
	UserID   types.String `tfsdk:"user_id"`
	TeamRole types.String `tfsdk:"team_role"`
}

func NewTeamMembersResource() resource.Resource { return &teamMembersResource{} }

func (r *teamMembersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_members"
}

func (r *teamMembersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	defaultedReplace := []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Authoritatively manages the complete membership of one team. Members added outside Terraform are removed on the next apply.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"organization": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: defaultedReplace, Description: "Organization slug; defaults to the provider organization."},
			"team_id":      schema.StringAttribute{Required: true, PlanModifiers: replace},
			"member": schema.SetNestedAttribute{Required: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"user_id":   schema.StringAttribute{Required: true},
				"team_role": schema.StringAttribute{Required: true, Description: "team_admin or team_member."},
			}}},
		},
	}
}

func (r *teamMembersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResource(req, resp, &r.provider)
}

func (r *teamMembersResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config teamMembersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	seen := make(map[string]struct{}, len(config.Members))
	for i, member := range config.Members {
		if member.UserID.IsUnknown() || member.UserID.IsNull() {
			continue
		}
		userID := member.UserID.ValueString()
		if !member.TeamRole.IsUnknown() && !member.TeamRole.IsNull() && member.TeamRole.ValueString() != "team_admin" && member.TeamRole.ValueString() != "team_member" {
			resp.Diagnostics.AddAttributeError(path.Root("member"), "Invalid team role", fmt.Sprintf("Member %d must use team_admin or team_member.", i))
		}
		if _, exists := seen[userID]; exists {
			resp.Diagnostics.AddAttributeError(path.Root("member"), "Duplicate team member", fmt.Sprintf("User %q is configured more than once (duplicate at member %d).", userID, i))
		}
		seen[userID] = struct{}{}
	}
}

func (r *teamMembersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan teamMembersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	org, err := organization(plan.Organization, r.provider.DefaultOrganization)
	if err != nil {
		resp.Diagnostics.AddError("Missing organization", err.Error())
		return
	}
	plan.ID = types.StringValue(plan.TeamID.ValueString())
	plan.Organization = types.StringValue(org)
	if err := r.reconcile(ctx, org, plan.TeamID.ValueString(), plan.Members); err != nil {
		apiDiagnostic(&resp.Diagnostics, "reconcile authoritative team membership", err)
		r.refreshState(ctx, &plan, &resp.State, &resp.Diagnostics)
		return
	}
	r.refreshState(ctx, &plan, &resp.State, &resp.Diagnostics)
}

func (r *teamMembersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state teamMembersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	members, err := r.provider.Client.ListTeamMembers(ctx, state.Organization.ValueString(), state.TeamID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		apiDiagnostic(&resp.Diagnostics, "read team membership", err)
		return
	}
	state.Members = flattenTeamMembers(members)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *teamMembersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan teamMembersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.reconcile(ctx, plan.Organization.ValueString(), plan.TeamID.ValueString(), plan.Members); err != nil {
		apiDiagnostic(&resp.Diagnostics, "reconcile authoritative team membership", err)
		r.refreshState(ctx, &plan, &resp.State, &resp.Diagnostics)
		return
	}
	r.refreshState(ctx, &plan, &resp.State, &resp.Diagnostics)
}

func (r *teamMembersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state teamMembersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	members, err := r.provider.Client.ListTeamMembers(ctx, state.Organization.ValueString(), state.TeamID.ValueString())
	if client.IsNotFound(err) {
		return
	}
	if err != nil {
		apiDiagnostic(&resp.Diagnostics, "read team membership before delete", err)
		return
	}
	for _, member := range members {
		err = r.provider.Client.RemoveTeamMember(ctx, state.Organization.ValueString(), state.TeamID.ValueString(), member.UserID)
		if err != nil && !client.IsNotFound(err) {
			apiDiagnostic(&resp.Diagnostics, "remove team member during authoritative membership delete", err)
			return
		}
	}
}

func (r *teamMembersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	org, teamID, err := parseImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected <organization-slug>/<team-id>.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), org)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), teamID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), teamID)...)
}

func (r *teamMembersResource) reconcile(ctx context.Context, org, teamID string, desired []teamMemberModel) error {
	current, err := r.provider.Client.ListTeamMembers(ctx, org, teamID)
	if err != nil {
		return err
	}
	currentByUser := make(map[string]client.TeamMember, len(current))
	for _, member := range current {
		currentByUser[member.UserID] = member
	}
	desiredByUser := make(map[string]string, len(desired))
	for _, member := range desired {
		userID, role := member.UserID.ValueString(), member.TeamRole.ValueString()
		desiredByUser[userID] = role
		stored, exists := currentByUser[userID]
		switch {
		case !exists:
			if _, err := r.provider.Client.AddTeamMember(ctx, org, teamID, map[string]any{"user_id": userID, "team_role": role}); err != nil {
				return err
			}
		case stored.TeamRole != role:
			if _, err := r.provider.Client.UpdateTeamMember(ctx, org, teamID, userID, map[string]any{"team_role": role}); err != nil {
				return err
			}
		}
	}
	for _, member := range current {
		if _, keep := desiredByUser[member.UserID]; keep {
			continue
		}
		if err := r.provider.Client.RemoveTeamMember(ctx, org, teamID, member.UserID); err != nil && !client.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (r *teamMembersResource) refreshState(ctx context.Context, model *teamMembersModel, state *tfsdk.State, diagnostics *diag.Diagnostics) {
	members, err := r.provider.Client.ListTeamMembers(ctx, model.Organization.ValueString(), model.TeamID.ValueString())
	if err != nil {
		apiDiagnostic(diagnostics, "refresh team membership", err)
		return
	}
	model.Members = flattenTeamMembers(members)
	diagnostics.Append(state.Set(ctx, model)...)
}

func flattenTeamMembers(members []client.TeamMember) []teamMemberModel {
	sort.Slice(members, func(i, j int) bool { return members[i].UserID < members[j].UserID })
	out := make([]teamMemberModel, len(members))
	for i, member := range members {
		out[i] = teamMemberModel{UserID: types.StringValue(member.UserID), TeamRole: types.StringValue(member.TeamRole)}
	}
	return out
}

var _ resource.ResourceWithImportState = (*teamMembersResource)(nil)
var _ resource.ResourceWithValidateConfig = (*teamMembersResource)(nil)

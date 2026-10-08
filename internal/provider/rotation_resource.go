package provider

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/dkurasov/terraform-provider-incidentgarden/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	rotationRestrictionWindowTypes = map[string]attr.Type{
		"start_hour": types.Int64Type,
		"start_min":  types.Int64Type,
		"end_hour":   types.Int64Type,
		"end_min":    types.Int64Type,
		"start_day":  types.StringType,
		"end_day":    types.StringType,
	}
	rotationTimeRestrictionTypes = map[string]attr.Type{
		"type":         types.StringType,
		"restrictions": types.ListType{ElemType: types.ObjectType{AttrTypes: rotationRestrictionWindowTypes}},
	}
)

type rotationResource struct{ provider *providerData }

type rotationModel struct {
	ID              types.String               `tfsdk:"id"`
	Organization    types.String               `tfsdk:"organization"`
	ScheduleID      types.String               `tfsdk:"schedule_id"`
	Name            types.String               `tfsdk:"name"`
	Type            types.String               `tfsdk:"type"`
	Length          types.Int64                `tfsdk:"length"`
	StartDate       types.String               `tfsdk:"start_date"`
	EndDate         types.String               `tfsdk:"end_date"`
	TimeRestriction types.Object               `tfsdk:"time_restriction"`
	Participants    []rotationParticipantModel `tfsdk:"participant"`
	CreatedAt       types.String               `tfsdk:"created_at"`
	UpdatedAt       types.String               `tfsdk:"updated_at"`
}

type rotationParticipantModel struct {
	ID              types.String `tfsdk:"id"`
	Position        types.Int64  `tfsdk:"position"`
	ParticipantType types.String `tfsdk:"participant_type"`
	UserID          types.String `tfsdk:"user_id"`
	TeamID          types.String `tfsdk:"team_id"`
}

func NewRotationResource() resource.Resource { return &rotationResource{} }

func (r *rotationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rotation"
}

func (r *rotationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "An ordered on-call rotation within a schedule.",
		Attributes: mergeAttributes(identityAttributes(), map[string]schema.Attribute{
			"schedule_id": schema.StringAttribute{Required: true, PlanModifiers: replace},
			"name":        schema.StringAttribute{Required: true},
			"type":        schema.StringAttribute{Required: true},
			"length":      schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}},
			"start_date":  schema.StringAttribute{Required: true},
			"end_date":    schema.StringAttribute{Optional: true},
			"time_restriction": schema.SingleNestedAttribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Object{objectplanmodifier.RequiresReplaceIf(replaceWhenClearingRotationRestriction, "Removing a time restriction replaces the rotation because the hosted API does not currently clear it in place.", "Removing a time restriction replaces the rotation because the hosted API does not currently clear it in place.")}, Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{Required: true, Description: "Must be weekday_and_time."},
				"restrictions": schema.ListNestedAttribute{Required: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"start_hour": schema.Int64Attribute{Required: true}, "start_min": schema.Int64Attribute{Required: true},
					"end_hour": schema.Int64Attribute{Required: true}, "end_min": schema.Int64Attribute{Required: true},
					"start_day": schema.StringAttribute{Required: true}, "end_day": schema.StringAttribute{Required: true},
				}}},
			}},
			"participant": schema.ListNestedAttribute{Required: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{Computed: true}, "position": schema.Int64Attribute{Computed: true},
				"participant_type": schema.StringAttribute{Required: true, Description: "user, team, or none."},
				"user_id":          schema.StringAttribute{Optional: true}, "team_id": schema.StringAttribute{Optional: true},
			}}},
		}),
	}
}

func replaceWhenClearingRotationRestriction(_ context.Context, req planmodifier.ObjectRequest, resp *objectplanmodifier.RequiresReplaceIfFuncResponse) {
	resp.RequiresReplace = req.ConfigValue.IsNull() && !req.StateValue.IsNull() && !req.StateValue.IsUnknown()
}

func (r *rotationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureResource(req, resp, &r.provider)
}

func (r *rotationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config rotationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(config.Participants) == 0 || len(config.Participants) > 50 {
		resp.Diagnostics.AddAttributeError(path.Root("participant"), "Invalid rotation participant count", "Configure between 1 and 50 participant positions.")
	}
	if !config.Type.IsUnknown() && !config.Type.IsNull() {
		switch config.Type.ValueString() {
		case "daily", "weekly", "hourly":
		default:
			resp.Diagnostics.AddAttributeError(path.Root("type"), "Invalid rotation type", "Use daily, weekly, or hourly.")
		}
	}
	if !config.Length.IsUnknown() && !config.Length.IsNull() && (config.Length.ValueInt64() < 1 || config.Length.ValueInt64() > 365) {
		resp.Diagnostics.AddAttributeError(path.Root("length"), "Invalid rotation length", "Configure a length from 1 through 365.")
	}
	start, startOK := validateRotationDate(path.Root("start_date"), config.StartDate, &resp.Diagnostics)
	end, endOK := validateRotationDate(path.Root("end_date"), config.EndDate, &resp.Diagnostics)
	if startOK && endOK && !end.After(start) {
		resp.Diagnostics.AddAttributeError(path.Root("end_date"), "Invalid rotation end date", "end_date must be later than start_date.")
	}
	for i, participant := range config.Participants {
		valid, known := validRotationParticipant(participant)
		if !known {
			continue
		}
		if !valid {
			resp.Diagnostics.AddAttributeError(path.Root("participant").AtListIndex(i), "Invalid rotation participant", "user requires only user_id, team requires only team_id, and none requires neither reference.")
		}
	}
	if !config.Type.IsUnknown() && config.Type.ValueString() == "hourly" && configured(config.TimeRestriction) {
		resp.Diagnostics.AddAttributeError(path.Root("time_restriction"), "Hourly rotation cannot have a time restriction", "Remove time_restriction or use a daily/weekly rotation.")
	}
	validateRotationRestriction(config.TimeRestriction, &resp.Diagnostics)
}

func validRotationParticipant(participant rotationParticipantModel) (bool, bool) {
	if participant.ParticipantType.IsUnknown() || participant.UserID.IsUnknown() || participant.TeamID.IsUnknown() {
		return false, false
	}
	hasUser, hasTeam := configured(participant.UserID), configured(participant.TeamID)
	switch participant.ParticipantType.ValueString() {
	case "user":
		return hasUser && !hasTeam, true
	case "team":
		return hasTeam && !hasUser, true
	case "none":
		return !hasUser && !hasTeam, true
	default:
		return false, true
	}
}

func validateRotationRestriction(restriction types.Object, diagnostics *diag.Diagnostics) {
	if !configured(restriction) {
		return
	}
	attributes := restriction.Attributes()
	restrictionType, ok := attributes["type"].(types.String)
	if ok && !restrictionType.IsNull() && !restrictionType.IsUnknown() && restrictionType.ValueString() != "weekday_and_time" {
		diagnostics.AddAttributeError(path.Root("time_restriction").AtName("type"), "Invalid time restriction type", "Use weekday_and_time.")
	}
	windows, ok := attributes["restrictions"].(types.List)
	if !ok || windows.IsNull() || windows.IsUnknown() {
		return
	}
	if len(windows.Elements()) > 20 {
		diagnostics.AddAttributeError(path.Root("time_restriction").AtName("restrictions"), "Too many time restriction windows", "Configure at most 20 windows.")
	}
	validDays := map[string]bool{"monday": true, "tuesday": true, "wednesday": true, "thursday": true, "friday": true, "saturday": true, "sunday": true}
	for index, element := range windows.Elements() {
		window, ok := element.(types.Object)
		if !ok || window.IsNull() || window.IsUnknown() {
			continue
		}
		values := window.Attributes()
		windowPath := path.Root("time_restriction").AtName("restrictions").AtListIndex(index)
		for _, field := range []string{"start_hour", "end_hour"} {
			value := values[field].(types.Int64)
			if !value.IsNull() && !value.IsUnknown() && (value.ValueInt64() < 0 || value.ValueInt64() > 23) {
				diagnostics.AddAttributeError(windowPath.AtName(field), "Invalid restriction hour", "Hours must be from 0 through 23.")
			}
		}
		for _, field := range []string{"start_min", "end_min"} {
			value := values[field].(types.Int64)
			if !value.IsNull() && !value.IsUnknown() && (value.ValueInt64() < 0 || value.ValueInt64() > 59) {
				diagnostics.AddAttributeError(windowPath.AtName(field), "Invalid restriction minute", "Minutes must be from 0 through 59.")
			}
		}
		for _, field := range []string{"start_day", "end_day"} {
			value := values[field].(types.String)
			if !value.IsNull() && !value.IsUnknown() && !validDays[value.ValueString()] {
				diagnostics.AddAttributeError(windowPath.AtName(field), "Invalid restriction day", "Use a lowercase weekday name from monday through sunday.")
			}
		}
		startDay, endDay := values["start_day"].(types.String), values["end_day"].(types.String)
		startHour, endHour := values["start_hour"].(types.Int64), values["end_hour"].(types.Int64)
		startMinute, endMinute := values["start_min"].(types.Int64), values["end_min"].(types.Int64)
		if configured(startDay) && configured(endDay) && configured(startHour) && configured(endHour) && configured(startMinute) && configured(endMinute) &&
			startDay.ValueString() == endDay.ValueString() && endHour.ValueInt64()*60+endMinute.ValueInt64() <= startHour.ValueInt64()*60+startMinute.ValueInt64() {
			diagnostics.AddAttributeError(windowPath, "Invalid same-day restriction window", "For a same-day window, the end time must be later than the start time; use different days for an overnight window.")
		}
	}
}

func validateRotationDate(attributePath path.Path, value types.String, diagnostics *diag.Diagnostics) (time.Time, bool) {
	if value.IsNull() || value.IsUnknown() {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value.ValueString())
	if err != nil {
		diagnostics.AddAttributeError(attributePath, "Invalid rotation date", "Use an RFC 3339 date-time.")
		return time.Time{}, false
	}
	if parsed.Year() < 2020 || parsed.Year() > 2099 {
		diagnostics.AddAttributeError(attributePath, "Rotation date outside supported range", "Use a date-time in calendar years 2020 through 2099.")
		return time.Time{}, false
	}
	return parsed, true
}

func rotationBody(model rotationModel, update bool) map[string]any {
	body := map[string]any{
		"name": model.Name.ValueString(), "type": model.Type.ValueString(), "start_date": model.StartDate.ValueString(),
		"participants": expandRotationParticipants(model.Participants),
	}
	if !model.Length.IsNull() && !model.Length.IsUnknown() {
		body["length"] = model.Length.ValueInt64()
	}
	if update && model.EndDate.IsNull() {
		body["end_date"] = nil
	} else if !model.EndDate.IsNull() && !model.EndDate.IsUnknown() {
		body["end_date"] = model.EndDate.ValueString()
	}
	if update && model.TimeRestriction.IsNull() {
		body["time_restriction"] = nil
	} else if !model.TimeRestriction.IsNull() && !model.TimeRestriction.IsUnknown() {
		body["time_restriction"] = objectValue(model.TimeRestriction)
	}
	return body
}

func expandRotationParticipants(participants []rotationParticipantModel) []map[string]any {
	out := make([]map[string]any, len(participants))
	for i, participant := range participants {
		out[i] = map[string]any{"participant_type": participant.ParticipantType.ValueString()}
		if userID := stringPointer(participant.UserID); userID != nil {
			out[i]["user_id"] = *userID
		}
		if teamID := stringPointer(participant.TeamID); teamID != nil {
			out[i]["team_id"] = *teamID
		}
	}
	return out
}

func (r *rotationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan rotationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	org, err := organization(plan.Organization, r.provider.DefaultOrganization)
	if err != nil {
		resp.Diagnostics.AddError("Missing organization", err.Error())
		return
	}
	remote, err := r.provider.Client.CreateRotation(ctx, org, plan.ScheduleID.ValueString(), rotationBody(plan, false))
	if err != nil {
		apiDiagnostic(&resp.Diagnostics, "create rotation", err)
		return
	}
	plan.apply(org, remote)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *rotationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state rotationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	schedule, err := r.provider.Client.GetScheduleDetail(ctx, state.Organization.ValueString(), state.ScheduleID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		apiDiagnostic(&resp.Diagnostics, "read schedule for rotation", err)
		return
	}
	for _, rotation := range schedule.Rotations {
		if rotation.ID == state.ID.ValueString() {
			state.apply(state.Organization.ValueString(), rotation)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *rotationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan rotationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := r.provider.Client.UpdateRotation(ctx, plan.Organization.ValueString(), plan.ScheduleID.ValueString(), plan.ID.ValueString(), rotationBody(plan, true))
	if err != nil {
		apiDiagnostic(&resp.Diagnostics, "update rotation", err)
		return
	}
	plan.apply(plan.Organization.ValueString(), remote)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *rotationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state rotationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.provider.Client.DeleteRotation(ctx, state.Organization.ValueString(), state.ScheduleID.ValueString(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		apiDiagnostic(&resp.Diagnostics, "delete rotation", err)
	}
}

func (r *rotationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected <organization-slug>/<schedule-id>/<rotation-id>.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("schedule_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func (model *rotationModel) apply(org string, rotation client.Rotation) {
	model.ID = types.StringValue(rotation.ID)
	model.Organization = types.StringValue(org)
	model.ScheduleID = types.StringValue(rotation.ScheduleID)
	model.Name = types.StringValue(rotation.Name)
	model.Type = types.StringValue(rotation.Type)
	model.Length = types.Int64Value(rotation.Length)
	model.StartDate = types.StringValue(rotation.StartDate)
	model.EndDate = nullableString(rotation.EndDate)
	if rotation.TimeRestriction == nil {
		model.TimeRestriction = types.ObjectNull(rotationTimeRestrictionTypes)
	} else {
		raw, _ := json.Marshal(rotation.TimeRestriction)
		model.TimeRestriction, _ = objectFromJSON(raw, rotationTimeRestrictionTypes)
	}
	model.Participants = make([]rotationParticipantModel, len(rotation.Participants))
	sort.SliceStable(rotation.Participants, func(i, j int) bool { return rotation.Participants[i].Position < rotation.Participants[j].Position })
	for i, participant := range rotation.Participants {
		model.Participants[i] = rotationParticipantModel{
			ID: types.StringValue(participant.ID), Position: types.Int64Value(participant.Position), ParticipantType: types.StringValue(participant.ParticipantType),
			UserID: nullableString(participant.UserID), TeamID: nullableString(participant.TeamID),
		}
	}
	model.CreatedAt = types.StringValue(rotation.CreatedAt)
	model.UpdatedAt = types.StringValue(rotation.UpdatedAt)
}

var _ resource.ResourceWithImportState = (*rotationResource)(nil)
var _ resource.ResourceWithValidateConfig = (*rotationResource)(nil)

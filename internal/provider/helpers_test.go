package provider

import (
	"context"
	"testing"

	"github.com/dkurasov/terraform-provider-incidentgarden/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestProviderSchemaIsValid(t *testing.T) {
	server := providerserver.NewProtocol6(New("test")())()
	response, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range response.Diagnostics {
		if diagnostic.Severity == tfprotov6.DiagnosticSeverityError {
			t.Fatalf("invalid provider schema: %s: %s", diagnostic.Summary, diagnostic.Detail)
		}
	}
}

func TestParseImportID(t *testing.T) {
	org, id, err := parseImportID("acme/550e8400-e29b-41d4-a716-446655440000")
	if err != nil || org != "acme" || id == "" {
		t.Fatalf("unexpected result %q %q %v", org, id, err)
	}
	for _, value := range []string{"", "acme", "/id", "acme/", "a/b/c"} {
		if _, _, err := parseImportID(value); err == nil {
			t.Errorf("expected %q to fail", value)
		}
	}
}

func TestCanonicalJSON(t *testing.T) {
	got, err := canonicalJSON([]byte(`{ "b": 2, "a": 1 }`), "{}")
	if err != nil || got.ValueString() != `{"a":1,"b":2}` {
		t.Fatalf("got %q, %v", got.ValueString(), err)
	}
}

func TestJSONStringSemanticEquality(t *testing.T) {
	left := jsonStringValue{StringValue: types.StringValue(`{"count":1,"nested":{"a":true}}`)}
	right := jsonStringValue{StringValue: types.StringValue("{\n  \"nested\": {\"a\": true}, \"count\": 1.0\n}")}
	equal, diags := left.StringSemanticEquals(context.Background(), right)
	if diags.HasError() || !equal {
		t.Fatalf("expected JSON values to be semantically equal: %v", diags)
	}
}

func TestDeactivateAfterCreate(t *testing.T) {
	if !deactivateAfterCreate(types.BoolValue(false)) {
		t.Fatal("explicit false must trigger post-create deactivation")
	}
	for _, value := range []types.Bool{types.BoolValue(true), types.BoolNull(), types.BoolUnknown()} {
		if deactivateAfterCreate(value) {
			t.Fatalf("unexpected deactivation for %v", value)
		}
	}
}

func TestTypedSeverityMappingPreservesEmptySourceField(t *testing.T) {
	mapping := objectFromMap(map[string]any{"source_field": "", "default": "warning"}, severityMappingTypes)
	body, err := integrationBody(integrationModel{SeverityMapping: mapping}, true, true)
	if err != nil {
		t.Fatal(err)
	}
	got := body["severity_mapping"].(map[string]any)
	if source, ok := got["source_field"]; !ok || source != "" {
		t.Fatalf("source_field was not preserved: %#v", got)
	}
}

func TestTypedPolicyAction(t *testing.T) {
	action := objectFromMap(map[string]any{"type": "add_labels", "value": map[string]any{"labels": map[string]any{"managed": "true"}}}, policyActionTypes)
	body, err := policyBody(integrationPolicyModel{Action: action}, true, false, true)
	if err != nil {
		t.Fatal(err)
	}
	got := body["actions"].([]any)[0].(map[string]any)
	if got["type"] != "add_labels" {
		t.Fatalf("unexpected action: %#v", got)
	}
}

func TestEscalationOffsetsCanBeStableButNotDescending(t *testing.T) {
	steps := []escalationStepModel{
		{ActivationOffsetMinutes: types.Int64Value(0)},
		{ActivationOffsetMinutes: types.Int64Value(0)},
		{ActivationOffsetMinutes: types.Int64Value(30)},
	}
	if index := firstDescendingStep(steps); index != -1 {
		t.Fatalf("stable equal offsets should be accepted, got index %d", index)
	}
	steps[2].ActivationOffsetMinutes = types.Int64Value(10)
	steps[1].ActivationOffsetMinutes = types.Int64Value(20)
	if index := firstDescendingStep(steps); index != 2 {
		t.Fatalf("descending step index = %d, want 2", index)
	}
}

func TestIntegrationReadPreservesOneTimeAPIKey(t *testing.T) {
	model := integrationModel{APIKey: types.StringValue("secret-returned-on-create")}
	model.apply("acme", client.Integration{ID: "id", Name: "name", Type: "alertmanager", TeamID: "team", EscalationPolicyID: "policy"})
	if got := model.APIKey.ValueString(); got != "secret-returned-on-create" {
		t.Fatalf("API key was not preserved: %q", got)
	}
	model.apply("acme", client.Integration{ID: "id", Name: "name", Type: "alertmanager", TeamID: "team", EscalationPolicyID: "policy", APIKey: "new-secret"})
	if got := model.APIKey.ValueString(); got != "new-secret" {
		t.Fatalf("creation API key was not stored: %q", got)
	}
}

func TestIntegrationDeactivatePreservesCreateOnlyAPIKey(t *testing.T) {
	created := client.Integration{APIKey: "secret-returned-on-create", APIKeyPrefix: "ig_123"}
	updated := client.Integration{ID: "id", IsActive: false}
	preserveCreateOnlyIntegrationFields(created, &updated)
	if updated.APIKey != created.APIKey || updated.APIKeyPrefix != created.APIKeyPrefix {
		t.Fatalf("create-only key fields were lost: %#v", updated)
	}
}

func TestRotationBodyPreservesOrderAndClearsNullableFields(t *testing.T) {
	model := rotationModel{
		Name: types.StringValue("Primary"), Type: types.StringValue("daily"), Length: types.Int64Value(1),
		StartDate: types.StringValue("2026-01-01T00:00:00Z"), EndDate: types.StringNull(), TimeRestriction: types.ObjectNull(rotationTimeRestrictionTypes),
		Participants: []rotationParticipantModel{
			{ParticipantType: types.StringValue("none")},
			{ParticipantType: types.StringValue("user"), UserID: types.StringValue("user-1")},
		},
	}
	body := rotationBody(model, true)
	participants := body["participants"].([]map[string]any)
	if participants[0]["participant_type"] != "none" || participants[1]["user_id"] != "user-1" {
		t.Fatalf("participant order changed: %#v", participants)
	}
	if value, present := body["end_date"]; !present || value != nil {
		t.Fatalf("end_date must be explicitly cleared: %#v", body)
	}
	if value, present := body["time_restriction"]; !present || value != nil {
		t.Fatalf("time_restriction must be explicitly cleared: %#v", body)
	}
	createBody := rotationBody(model, false)
	if _, present := createBody["end_date"]; present {
		t.Fatalf("create unexpectedly sent null end_date: %#v", createBody)
	}
}

func TestFlattenTeamMembersIsStable(t *testing.T) {
	members := flattenTeamMembers([]client.TeamMember{{UserID: "b", TeamRole: "team_member"}, {UserID: "a", TeamRole: "team_admin"}})
	if members[0].UserID.ValueString() != "a" || members[1].UserID.ValueString() != "b" {
		t.Fatalf("members not sorted: %#v", members)
	}
}

func TestRotationReadOrdersParticipantsByPosition(t *testing.T) {
	model := rotationModel{}
	model.apply("acme", client.Rotation{
		ID: "rotation-1", ScheduleID: "schedule-1", Name: "Primary", Type: "daily", Length: 1, StartDate: "2026-01-01T00:00:00Z",
		Participants: []client.RotationParticipant{
			{ID: "second", ParticipantType: "none", Position: 1},
			{ID: "first", ParticipantType: "none", Position: 0},
		},
	})
	if model.Participants[0].ID.ValueString() != "first" || model.Participants[1].ID.ValueString() != "second" {
		t.Fatalf("participants not ordered by position: %#v", model.Participants)
	}
}

func TestRotationRestrictionRejectsSameDayOvernight(t *testing.T) {
	restriction := objectFromMap(map[string]any{
		"type": "weekday_and_time",
		"restrictions": []any{map[string]any{
			"start_hour": float64(22), "start_min": float64(0), "end_hour": float64(6), "end_min": float64(0),
			"start_day": "monday", "end_day": "monday",
		}},
	}, rotationTimeRestrictionTypes)
	var diagnostics diag.Diagnostics
	validateRotationRestriction(restriction, &diagnostics)
	if !diagnostics.HasError() {
		t.Fatal("expected same-day overnight restriction to fail")
	}
}

func TestRotationValidationDefersUnknownParticipantReference(t *testing.T) {
	participant := rotationParticipantModel{ParticipantType: types.StringValue("user"), UserID: types.StringUnknown(), TeamID: types.StringNull()}
	if valid, known := validRotationParticipant(participant); valid || known {
		t.Fatalf("unknown reference returned valid=%t known=%t", valid, known)
	}
}

func TestRemovingRotationRestrictionRequiresReplacement(t *testing.T) {
	request := planmodifier.ObjectRequest{
		ConfigValue: types.ObjectNull(rotationTimeRestrictionTypes),
		StateValue: objectFromMap(map[string]any{
			"type": "weekday_and_time",
			"restrictions": []any{map[string]any{
				"start_hour": float64(9), "start_min": float64(0), "end_hour": float64(18), "end_min": float64(0),
				"start_day": "monday", "end_day": "monday",
			}},
		}, rotationTimeRestrictionTypes),
	}
	response := &objectplanmodifier.RequiresReplaceIfFuncResponse{}
	replaceWhenClearingRotationRestriction(context.Background(), request, response)
	if !response.RequiresReplace {
		t.Fatal("removing a hosted time restriction must replace the rotation")
	}
}

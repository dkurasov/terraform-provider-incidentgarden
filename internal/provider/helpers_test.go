package provider

import (
	"testing"

	"github.com/dkurasov/incidentgarden-terraform-provider/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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

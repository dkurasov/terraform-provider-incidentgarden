package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	acctest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTeamBasic(t *testing.T) {
	org := os.Getenv("INCIDENTGARDEN_TEST_ORGANIZATION")
	if org == "" {
		t.Skip("INCIDENTGARDEN_TEST_ORGANIZATION is required")
	}
	acctest.Test(t, acctest.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"incidentgarden": providerserver.NewProtocol6WithError(New("test")())},
		Steps: []acctest.TestStep{
			{Config: testTeamConfig(org, "Terraform Team"), Check: acctest.ComposeAggregateTestCheckFunc(acctest.TestCheckResourceAttr("incidentgarden_team.test", "name", "Terraform Team"), acctest.TestCheckResourceAttrSet("incidentgarden_team.test", "id"))},
			{Config: testTeamConfig(org, "Terraform Team Updated"), Check: acctest.TestCheckResourceAttr("incidentgarden_team.test", "name", "Terraform Team Updated")},
			{ResourceName: "incidentgarden_team.test", ImportState: true, ImportStateIdPrefix: org + "/", ImportStateVerify: true},
		},
	})
}

func TestAccTopologyBasic(t *testing.T) {
	org := os.Getenv("INCIDENTGARDEN_TEST_ORGANIZATION")
	if org == "" {
		t.Skip("INCIDENTGARDEN_TEST_ORGANIZATION is required")
	}

	acctest.Test(t, acctest.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"incidentgarden": providerserver.NewProtocol6WithError(New("test")())},
		Steps: []acctest.TestStep{
			{
				Config: testTopologyConfig(org, "Initial", true),
				Check: acctest.ComposeAggregateTestCheckFunc(
					acctest.TestCheckResourceAttrSet("incidentgarden_schedule.test", "id"),
					acctest.TestCheckResourceAttr("incidentgarden_schedule.test", "name", "Terraform Schedule Initial"),
					acctest.TestCheckResourceAttrSet("incidentgarden_escalation_policy.test", "id"),
					acctest.TestCheckResourceAttr("incidentgarden_escalation_policy.test", "step.0.activation_offset_minutes", "0"),
					acctest.TestCheckResourceAttrSet("incidentgarden_integration.test", "id"),
					acctest.TestCheckResourceAttrSet("incidentgarden_integration.test", "api_key"),
					acctest.TestCheckResourceAttrSet("incidentgarden_integration_policy.test", "id"),
					acctest.TestCheckResourceAttr("incidentgarden_integration_policy.test", "position", "0"),
				),
			},
			{
				Config: testTopologyConfig(org, "Updated", false),
				Check: acctest.ComposeAggregateTestCheckFunc(
					acctest.TestCheckResourceAttr("incidentgarden_schedule.test", "name", "Terraform Schedule Updated"),
					acctest.TestCheckResourceAttr("incidentgarden_escalation_policy.test", "name", "Terraform Escalation Updated"),
					acctest.TestCheckResourceAttr("incidentgarden_integration.test", "name", "Terraform Integration Updated"),
					acctest.TestCheckResourceAttr("incidentgarden_integration.test", "is_active", "false"),
					acctest.TestCheckResourceAttr("incidentgarden_integration_policy.test", "name", "Terraform Policy Updated"),
					acctest.TestCheckResourceAttr("incidentgarden_integration_policy.test", "is_active", "false"),
				),
			},
			{ResourceName: "incidentgarden_schedule.test", ImportState: true, ImportStateIdPrefix: org + "/", ImportStateVerify: true},
			{ResourceName: "incidentgarden_escalation_policy.test", ImportState: true, ImportStateIdPrefix: org + "/", ImportStateVerify: true},
			{ResourceName: "incidentgarden_integration.test", ImportState: true, ImportStateIdPrefix: org + "/", ImportStateVerify: true, ImportStateVerifyIgnore: []string{"api_key"}},
			{ResourceName: "incidentgarden_integration_policy.test", ImportState: true, ImportStateIdPrefix: org + "/", ImportStateVerify: true},
		},
	})
}

func testTeamConfig(org, name string) string {
	return fmt.Sprintf(`
provider "incidentgarden" {}

resource "incidentgarden_team" "test" {
  organization = %q
  name         = %q
}
`, org, name)
}

func testTopologyConfig(org, suffix string, active bool) string {
	return fmt.Sprintf(`
provider "incidentgarden" {}

resource "incidentgarden_team" "test" {
  organization = %[1]q
  name         = "Terraform Topology Team"
}

resource "incidentgarden_schedule" "test" {
  organization = %[1]q
  team_id      = incidentgarden_team.test.id
  name         = "Terraform Schedule %[2]s"
}

resource "incidentgarden_escalation_policy" "test" {
  organization = %[1]q
  team_id      = incidentgarden_team.test.id
  name         = "Terraform Escalation %[2]s"

  step = [{
    target_type               = "on_call_schedule"
    schedule_id               = incidentgarden_schedule.test.id
    activation_offset_minutes = 0
  }]
}

resource "incidentgarden_integration" "test" {
  organization         = %[1]q
  name                 = "Terraform Integration %[2]s"
  type                 = "generic_webhook"
  team_id              = incidentgarden_team.test.id
  escalation_policy_id = incidentgarden_escalation_policy.test.id
  is_active            = %[3]t
}

resource "incidentgarden_integration_policy" "test" {
  organization   = %[1]q
  team_id        = incidentgarden_team.test.id
  integration_id = incidentgarden_integration.test.id
  applies_to_all = false
  name           = "Terraform Policy %[2]s"
  position       = 0
  is_active      = %[3]t
  conditions_json = jsonencode({
    matchers = [{ key = "source", op = "exists" }]
  })
  action_json = jsonencode({ type = "drop" })
}
`, org, suffix, active)
}

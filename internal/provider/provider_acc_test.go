package provider

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/dkurasov/terraform-provider-incidentgarden/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	acctest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccTeamBasic(t *testing.T) {
	org := os.Getenv("INCIDENTGARDEN_TEST_ORGANIZATION")
	if org == "" {
		t.Skip("INCIDENTGARDEN_TEST_ORGANIZATION is required")
	}
	name := uniqueAcceptanceName("Terraform Team")
	acctest.Test(t, acctest.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"incidentgarden": providerserver.NewProtocol6WithError(New("test")())},
		CheckDestroy:             checkAllResourcesDestroyed,
		Steps: []acctest.TestStep{
			{Config: testTeamConfig(org, name), Check: acctest.ComposeAggregateTestCheckFunc(acctest.TestCheckResourceAttr("incidentgarden_team.test", "name", name), acctest.TestCheckResourceAttrSet("incidentgarden_team.test", "id"))},
			{Config: testTeamConfig(org, name+" Updated"), Check: acctest.TestCheckResourceAttr("incidentgarden_team.test", "name", name+" Updated")},
			{ResourceName: "incidentgarden_team.test", ImportState: true, ImportStateIdPrefix: org + "/", ImportStateVerify: true},
		},
	})
}

func TestAccTopologyBasic(t *testing.T) {
	if os.Getenv("INCIDENTGARDEN_RUN_TOPOLOGY_ACC") != "1" {
		t.Skip("INCIDENTGARDEN_RUN_TOPOLOGY_ACC=1 is required because the hosted API has a known integration cleanup defect")
	}
	org := os.Getenv("INCIDENTGARDEN_TEST_ORGANIZATION")
	if org == "" {
		t.Skip("INCIDENTGARDEN_TEST_ORGANIZATION is required")
	}
	name := uniqueAcceptanceName("Terraform Topology")

	acctest.Test(t, acctest.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){"incidentgarden": providerserver.NewProtocol6WithError(New("test")())},
		CheckDestroy:             checkAllResourcesDestroyed,
		Steps: []acctest.TestStep{
			{
				Config: testTopologyConfig(org, name, "Initial", true),
				Check: acctest.ComposeAggregateTestCheckFunc(
					acctest.TestCheckResourceAttrSet("incidentgarden_schedule.test", "id"),
					acctest.TestCheckResourceAttr("incidentgarden_schedule.test", "name", name+" Schedule Initial"),
					acctest.TestCheckResourceAttrSet("incidentgarden_escalation_policy.test", "id"),
					acctest.TestCheckResourceAttr("incidentgarden_escalation_policy.test", "step.0.activation_offset_minutes", "0"),
					acctest.TestCheckResourceAttrSet("incidentgarden_integration.test", "id"),
					acctest.TestCheckResourceAttrSet("incidentgarden_integration.test", "api_key"),
					acctest.TestCheckResourceAttrSet("incidentgarden_integration_policy.test", "id"),
					acctest.TestCheckResourceAttr("incidentgarden_integration_policy.test", "position", "0"),
				),
			},
			{
				Config: testTopologyConfig(org, name, "Updated", false),
				Check: acctest.ComposeAggregateTestCheckFunc(
					acctest.TestCheckResourceAttr("incidentgarden_schedule.test", "name", name+" Schedule Updated"),
					acctest.TestCheckResourceAttr("incidentgarden_escalation_policy.test", "name", name+" Escalation Updated"),
					acctest.TestCheckResourceAttr("incidentgarden_integration.test", "name", name+" Integration Updated"),
					acctest.TestCheckResourceAttr("incidentgarden_integration.test", "is_active", "false"),
					acctest.TestCheckResourceAttr("incidentgarden_integration_policy.test", "name", name+" Policy Updated"),
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

func testTopologyConfig(org, name, suffix string, active bool) string {
	return fmt.Sprintf(`
provider "incidentgarden" {}

resource "incidentgarden_team" "test" {
  organization = %[1]q
  name         = "%[2]s Team"
}

resource "incidentgarden_schedule" "test" {
  organization = %[1]q
  team_id      = incidentgarden_team.test.id
  name         = "%[2]s Schedule %[3]s"
}

resource "incidentgarden_escalation_policy" "test" {
  organization = %[1]q
  team_id      = incidentgarden_team.test.id
  name         = "%[2]s Escalation %[3]s"

  step = [{
    target_type               = "on_call_schedule"
    schedule_id               = incidentgarden_schedule.test.id
    activation_offset_minutes = 0
  }]
}

resource "incidentgarden_integration" "test" {
  organization         = %[1]q
  name                 = "%[2]s Integration %[3]s"
  type                 = "generic_webhook"
  team_id              = incidentgarden_team.test.id
  escalation_policy_id = incidentgarden_escalation_policy.test.id
  is_active            = %[4]t
}

resource "incidentgarden_integration_policy" "test" {
  organization   = %[1]q
  team_id        = incidentgarden_team.test.id
  integration_id = incidentgarden_integration.test.id
  applies_to_all = false
  name           = "%[2]s Policy %[3]s"
  position       = 0
  is_active      = %[4]t
  conditions_json = jsonencode({
    matchers = [{ key = "source", op = "exists" }]
  })
  action_json = jsonencode({ type = "drop" })
}
`, org, name, suffix, active)
}

func uniqueAcceptanceName(prefix string) string {
	return fmt.Sprintf("%s %d", prefix, time.Now().UnixNano())
}

func checkAllResourcesDestroyed(state *terraform.State) error {
	apiClient, err := client.New(os.Getenv("INCIDENTGARDEN_ENDPOINT"), os.Getenv("INCIDENTGARDEN_TOKEN"), os.Getenv("INCIDENTGARDEN_CSRF_TOKEN"), nil)
	if err != nil {
		return err
	}
	for _, remote := range state.RootModule().Resources {
		organization := remote.Primary.Attributes["organization"]
		id := remote.Primary.ID
		var getErr error
		switch remote.Type {
		case "incidentgarden_team":
			_, getErr = apiClient.GetTeam(context.Background(), organization, id)
		case "incidentgarden_schedule":
			_, getErr = apiClient.GetSchedule(context.Background(), organization, id)
		case "incidentgarden_escalation_policy":
			_, getErr = apiClient.GetEscalationPolicy(context.Background(), organization, id)
		case "incidentgarden_integration":
			_, getErr = apiClient.GetIntegration(context.Background(), organization, id)
		case "incidentgarden_integration_policy":
			_, getErr = apiClient.GetIntegrationPolicy(context.Background(), organization, id)
		default:
			continue
		}
		if getErr == nil {
			return fmt.Errorf("%s %s still exists after destroy", remote.Type, id)
		}
		if !client.IsNotFound(getErr) {
			return fmt.Errorf("verify %s %s destruction: %w", remote.Type, id, getErr)
		}
	}
	return nil
}

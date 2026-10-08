terraform {
  required_providers {
    incidentgarden = {
      source = "dkurasov/incidentgarden"
    }
  }
}

provider "incidentgarden" {
  endpoint     = var.endpoint
  organization = var.organization
}

data "incidentgarden_members" "organization" {}

resource "incidentgarden_team" "manual_test" {
  name        = "Terraform Manual Test"
  description = "Safe to delete after manual provider testing"
}

resource "incidentgarden_schedule" "manual_test" {
  team_id     = incidentgarden_team.manual_test.id
  name        = "Terraform Manual Schedule"
  description = "Schedule created by the live Terraform example"
}

resource "incidentgarden_team_members" "manual_test" {
  team_id = incidentgarden_team.manual_test.id

  member = [{
    user_id   = data.incidentgarden_members.organization.by_email[var.on_call_email]
    team_role = "team_admin"
  }]
}

resource "incidentgarden_rotation" "manual_test" {
  schedule_id = incidentgarden_schedule.manual_test.id
  name        = "Terraform Manual Rotation"
  type        = "daily"
  length      = 1
  start_date  = "2026-01-01T09:00:00Z"

  participant = [{
    participant_type = "user"
    user_id          = data.incidentgarden_members.organization.by_email[var.on_call_email]
  }]

  depends_on = [incidentgarden_team_members.manual_test]
}

resource "incidentgarden_escalation_policy" "manual_test" {
  team_id     = incidentgarden_team.manual_test.id
  name        = "Terraform Manual Escalation"
  description = "Escalates immediately to the test schedule"

  step = [{
    target_type               = "on_call_schedule"
    schedule_id               = incidentgarden_schedule.manual_test.id
    activation_offset_minutes = 0
  }]

  depends_on = [incidentgarden_rotation.manual_test]
}

resource "incidentgarden_integration" "manual_test" {
  name                 = "Terraform Manual Webhook"
  type                 = "generic_webhook"
  team_id              = incidentgarden_team.manual_test.id
  escalation_policy_id = incidentgarden_escalation_policy.manual_test.id
  is_active            = true

  severity_mapping = {
    source_field = "severity"
    mapping = {
      critical = ["critical", "high"]
      warning  = ["warning", "medium"]
    }
    default = "warning"
  }
}

resource "incidentgarden_integration_policy" "manual_test" {
  team_id        = incidentgarden_team.manual_test.id
  integration_id = incidentgarden_integration.manual_test.id
  applies_to_all = false
  name           = "Drop explicitly ignored alerts"
  position       = 0
  is_active      = true

  conditions = {
    matchers = [{ key = "terraform_ignore", op = "eq", value = "true" }]
  }

  action = { type = "drop" }
}

output "team_id" {
  value = incidentgarden_team.manual_test.id
}

output "integration_id" {
  value = incidentgarden_integration.manual_test.id
}

output "integration_api_key" {
  value     = incidentgarden_integration.manual_test.api_key
  sensitive = true
}

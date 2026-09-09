terraform {
  required_providers {
    incidentgarden = {
      source = "dkurasov/incidentgarden"
    }
  }
}

provider "incidentgarden" {
  endpoint     = var.endpoint
  token        = var.token
  csrf_token   = var.csrf_token
  organization = var.organization
}

resource "incidentgarden_team" "platform" {
  name        = "Platform"
  description = "Platform operations"
}

resource "incidentgarden_schedule" "primary" {
  team_id     = incidentgarden_team.platform.id
  name        = "Primary"
  description = "Primary on-call schedule"
}

resource "incidentgarden_escalation_policy" "critical" {
  team_id = incidentgarden_team.platform.id
  name    = "Critical alerts"

  step = [{
    target_type               = "on_call_schedule"
    schedule_id               = incidentgarden_schedule.primary.id
    activation_offset_minutes = 0
  }]
}

resource "incidentgarden_integration" "alertmanager" {
  name                 = "Alertmanager"
  type                 = "alertmanager"
  team_id              = incidentgarden_team.platform.id
  escalation_policy_id = incidentgarden_escalation_policy.critical.id
}

resource "incidentgarden_integration_policy" "production" {
  team_id        = incidentgarden_team.platform.id
  integration_id = incidentgarden_integration.alertmanager.id
  applies_to_all = false
  name           = "Drop non-production alerts"
  position       = 0
  conditions_json = jsonencode({
    matchers = [{ key = "env", op = "neq", value = "production" }]
  })
  action_json = jsonencode({ type = "drop" })
}

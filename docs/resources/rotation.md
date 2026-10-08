# incidentgarden_rotation

Manages one ordered on-call rotation within a schedule. Participants may target
a user, target a team, or deliberately leave a `none` position in the circle.

```hcl
resource "incidentgarden_rotation" "primary" {
  schedule_id = incidentgarden_schedule.primary.id
  name        = "Primary"
  type        = "daily"
  length      = 1
  start_date  = "2026-01-01T09:00:00Z"

  participant = [{
    participant_type = "user"
    user_id          = data.incidentgarden_members.organization.by_email[var.on_call_email]
  }]
}
```

Import with `<organization-slug>/<schedule-id>/<rotation-id>`.

Removing `time_restriction` replaces the rotation. The contract permits an
in-place null clear, but hosted production retained the old restriction during
validation on 2026-10-08; replacement avoids a perpetual Terraform diff.

# incidentgarden_team_members

Authoritatively owns the complete member set of a team. Members absent from the
configuration are removed on apply, and all members are removed on destroy.
Use this resource only for a team exclusively managed by this Terraform state.

```hcl
resource "incidentgarden_team_members" "platform" {
  team_id = incidentgarden_team.platform.id
  member = [{
    user_id   = data.incidentgarden_members.organization.by_email[var.on_call_email]
    team_role = "team_admin"
  }]
}
```

Import with `<organization-slug>/<team-id>`.

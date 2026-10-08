# incidentgarden_members

Lists all organization members and maps exact email addresses to user IDs. The
calling account must be an organization administrator or owner.

```hcl
data "incidentgarden_members" "organization" {}

output "user_id" {
  value = data.incidentgarden_members.organization.by_email[var.on_call_email]
}
```

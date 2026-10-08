variable "endpoint" {
  type = string
}

variable "organization" {
  type = string
}

variable "on_call_email" {
  type        = string
  description = "Exact email address of the organization member who should receive alerts. The caller must be an organization admin or owner to resolve it."
}

variable "token" {
  type      = string
  sensitive = true
}

variable "csrf_token" {
  type      = string
  sensitive = true
}

variable "endpoint" {
  type        = string
  description = "Incident Garden API base URL."
  default     = "https://api.incidentgarden.ru"
}

variable "organization" {
  type        = string
  description = "Organization slug used for the manual test topology."
  default     = "david-tech-org"
}

variable "on_call_email" {
  type        = string
  description = "Exact email address of the organization member who should receive manual-test alerts."
}

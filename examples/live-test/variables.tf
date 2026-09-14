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

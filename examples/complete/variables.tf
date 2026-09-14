variable "endpoint" {
  type = string
}

variable "organization" {
  type = string
}

variable "token" {
  type      = string
  sensitive = true
}

variable "csrf_token" {
  type      = string
  sensitive = true
}

variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "account_id" {
  description = "Service account ID"
  type        = string
}

variable "display_name" {
  description = "Service account display name"
  type        = string
  default     = null
}

variable "roles" {
  description = "IAM roles to assign to the service account"
  type        = set(string)
  default     = []
}

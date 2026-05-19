variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "services" {
  description = "GCP APIs to enable"
  type        = list(string)
}

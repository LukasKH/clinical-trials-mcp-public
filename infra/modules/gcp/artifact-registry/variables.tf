variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP region for the repository"
  type        = string
}

variable "repository_id" {
  description = "Artifact Registry repository ID"
  type        = string
}

variable "description" {
  description = "Artifact Registry repository description"
  type        = string
  default     = ""
}

variable "format" {
  description = "Artifact Registry repository format"
  type        = string
  default     = "DOCKER"
}

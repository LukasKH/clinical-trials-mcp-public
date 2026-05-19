variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "Cloud Run region"
  type        = string
}

variable "service_name" {
  description = "Cloud Run service name"
  type        = string
}

variable "image" {
  description = "Bootstrap container image. App deployment owns later image rollouts."
  type        = string
  default     = "us-docker.pkg.dev/cloudrun/container/hello"
}

variable "service_account_email" {
  description = "Service account email for the service"
  type        = string
  default     = ""
}

variable "env_vars" {
  description = "Environment variables for the container"
  type        = map(string)
  default     = {}
}

variable "ingress" {
  description = "Cloud Run ingress setting"
  type        = string
  default     = "INGRESS_TRAFFIC_ALL"
}

variable "invoker_iam_disabled" {
  description = "Whether to disable the Cloud Run Invoker IAM check. When true, the service is publicly callable."
  type        = bool
  default     = false
}

variable "cpu" {
  description = "CPU limit for the main container"
  type        = string
  default     = "1"
}

variable "memory" {
  description = "Memory limit for the main container"
  type        = string
  default     = "512Mi"
}

variable "cpu_idle" {
  description = "Whether to allocate CPU only during request processing"
  type        = bool
  default     = true
}

variable "startup_cpu_boost" {
  description = "Whether to enable Cloud Run startup CPU boost for faster cold starts"
  type        = bool
  default     = false
}

variable "timeout_seconds" {
  description = "Request timeout in seconds"
  type        = number
  default     = 300
}

variable "concurrency" {
  description = "Maximum concurrent requests per instance"
  type        = number
  default     = 80
}

variable "min_instance_count" {
  description = "Minimum number of serving instances"
  type        = number
  default     = 0
}

variable "max_instance_count" {
  description = "Maximum number of serving instances"
  type        = number
  default     = 3
}

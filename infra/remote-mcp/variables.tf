variable "project_id" {
  description = "Google Cloud project that hosts the remote ClinicalTrials.gov MCP service."
  type        = string
}

variable "region" {
  description = "Cloud Run region for the remote MCP service."
  type        = string
  default     = "us-central1"
}

variable "service_name" {
  description = "Cloud Run service name for the remote MCP server."
  type        = string
  default     = "clinicaltrials-gov-mcp"
}

variable "artifact_repository_id" {
  description = "Artifact Registry repository for MCP container images."
  type        = string
  default     = "clinical-trials-mcp"
}

variable "runtime_service_account_id" {
  description = "Runtime service account ID. Must be 30 characters or fewer."
  type        = string
  default     = "trial-mcp-runtime"
}

variable "ci_deployer_service_account_id" {
  description = "Service account used by GitHub Actions to deploy Cloud Run revisions."
  type        = string
  default     = "trial-mcp-ci-deployer"
}

variable "terraform_deployer_service_account_email" {
  description = "Bootstrap-created service account email used by GitHub Actions to run Terraform apply."
  type        = string
}

variable "github_repository" {
  description = "GitHub repository allowed to deploy through Workload Identity Federation, in owner/name format."
  type        = string
}

variable "deploy_branch" {
  description = "Git branch allowed to deploy through Workload Identity Federation."
  type        = string
  default     = "main"
}

variable "github_actions_pool_id" {
  description = "Workload Identity Pool ID for GitHub Actions."
  type        = string
  default     = "github-actions"
}

variable "github_actions_provider_id" {
  description = "Workload Identity Pool Provider ID for GitHub Actions OIDC."
  type        = string
  default     = "github-actions"
}

variable "container_image" {
  description = "Bootstrap image for first Cloud Run creation. CI/CD owns real MCP image rollouts."
  type        = string
  default     = "us-docker.pkg.dev/cloudrun/container/hello"
}

variable "public_access_enabled" {
  description = "When true, make the MCP service publicly callable by disabling the Cloud Run Invoker IAM check. This repo intentionally defaults to public self-hosted MCP access."
  type        = bool
  default     = true
}

variable "clinical_trials_api_base_url" {
  description = "ClinicalTrials.gov API v2 base URL."
  type        = string
  default     = "https://clinicaltrials.gov/api/v2"
}

variable "eu_clinical_trials_api_base_url" {
  description = "EU Clinical Trials CTIS public API base URL."
  type        = string
  default     = "https://euclinicaltrials.eu/ctis-public-api"
}

variable "clinical_trials_request_timeout_seconds" {
  description = "Timeout for outbound clinical trial registry API requests."
  type        = number
  default     = 30
}

variable "clinical_trials_max_page_size" {
  description = "Maximum registry page size exposed by the MCP search tool."
  type        = number
  default     = 200
}

variable "startup_cpu_boost" {
  description = "Enable Cloud Run startup CPU boost to improve cold start latency."
  type        = bool
  default     = true
}

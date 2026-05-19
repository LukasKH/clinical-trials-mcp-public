output "service_name" {
  description = "Cloud Run service name."
  value       = module.cloud_run.service_name
}

output "service_account_email" {
  description = "Runtime service account used by the remote MCP service."
  value       = module.runtime_sa.email
}

output "artifact_repository_url" {
  description = "Artifact Registry repository URL for MCP images."
  value       = module.artifact_registry.repository_url
}

output "mcp_image_url" {
  description = "Expected MCP image URL for CI/CD rollout."
  value       = "${module.artifact_registry.repository_url}/${var.service_name}:latest"
}

output "service_uri" {
  description = "Base Cloud Run service URI."
  value       = module.cloud_run.service_uri
}

output "mcp_url" {
  description = "Streamable HTTP MCP endpoint URL."
  value       = "${module.cloud_run.service_uri}/mcp"
}

output "github_actions_workload_identity_provider" {
  description = "Workload Identity Provider resource name for GitHub Actions."
  value       = google_iam_workload_identity_pool_provider.github_actions.name
}

output "ci_deployer_service_account_email" {
  description = "Service account email that GitHub Actions impersonates for deployment."
  value       = module.ci_deployer_sa.email
}

output "terraform_deployer_service_account_email" {
  description = "Bootstrap-created service account email that GitHub Actions impersonates for Terraform apply."
  value       = var.terraform_deployer_service_account_email
}

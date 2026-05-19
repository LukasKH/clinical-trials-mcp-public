output "repository_id" {
  description = "Repository ID"
  value       = google_artifact_registry_repository.repo.repository_id
}

output "repository_name" {
  description = "Repository resource name"
  value       = google_artifact_registry_repository.repo.name
}

output "repository_url" {
  description = "Repository URL"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}"
}

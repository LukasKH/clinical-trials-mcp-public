output "enabled_services" {
  description = "List of enabled GCP services"
  value       = [for service in google_project_service.apis : service.service]
}

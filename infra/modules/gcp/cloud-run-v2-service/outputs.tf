output "service_name" {
  description = "Cloud Run service name"
  value       = google_cloud_run_v2_service.service.name
}

output "service_id" {
  description = "Cloud Run service resource ID"
  value       = google_cloud_run_v2_service.service.id
}

output "service_uri" {
  description = "Cloud Run service URL"
  value       = google_cloud_run_v2_service.service.uri
}

output "location" {
  description = "Cloud Run service location"
  value       = google_cloud_run_v2_service.service.location
}

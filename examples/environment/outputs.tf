# Output the environment ID
output "environment_id" {
  value = data.solacecloud_environment.environment.id
}

# Output the environment type
output "environment_type" {
  value = data.solacecloud_environment.environment.type
}

# Output the service ID
output "service_id" {
  value = solacecloud_service.broker_service.id
}

output "service_id" {
  value = solacecloud_service.broker_service.id
}

output "service_class_id" {
  value = solacecloud_service.broker_service.service_class_id
}

output "broker_details" {
  value = solacecloud_service.broker_service
  sensitive = true
}
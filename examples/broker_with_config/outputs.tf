output "broker_username" {
  value = solacecloud_service.broker_service.message_vpn.manager_management_credential.username
}

output "broker_password" {
  value = solacecloud_service.broker_service.message_vpn.manager_management_credential.password
  sensitive = true
}

output "broker_semp_url" {
  value = "https://${data.solacecloud_connection_endpoint_dns_names.broker1_dns.dns_names[0].dns_name}:${data.solacecloud_connection_endpoints.broker1_endpoints.endpoints[0].ports.management_tls.port}"
}

output "broker_username" {
  value = solacecloud_service.broker_service.message_vpn.manager_management_credential.username
}

output "broker_password" {
  value = solacecloud_service.broker_service.message_vpn.manager_management_credential.password
  sensitive = true
}

output "broker_semp_url" {
  value = "https://${solacecloud_service.broker_service.connection_endpoints[0].hostnames[0]}:${solacecloud_service.broker_service.connection_endpoints[0].ports.management_tls.port}"
}

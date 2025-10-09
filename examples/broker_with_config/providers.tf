terraform {
  required_providers {
    solacecloud = {
      source = "registry.terraform.io/solaceproducts/solacecloud"
    }
    solacebroker = {
      source  = "SolaceProducts/solacebroker"
      version = "1.1.0"
    }
  }
}

provider "solacecloud" {
  base_url             = "https://api.solace.cloud/"
  api_polling_interval = 30
}

data "solacecloud_connection_endpoints" "broker1_endpoints" {
  service_id = solacecloud_service.broker_service.id
}

data "solacecloud_connection_endpoint_dns_names" "broker1_dns" {
  service_id             = solacecloud_service.broker_service.id
  connection_endpoint_id = data.solacecloud_connection_endpoints.broker1_endpoints.endpoints[0].id
}

data "solacecloud_connection_endpoints" "broker2_endpoints" {
  service_id = solacecloud_service.broker_service2.id
}

data "solacecloud_connection_endpoint_dns_names" "broker2_dns" {
  service_id             = solacecloud_service.broker_service2.id
  connection_endpoint_id = data.solacecloud_connection_endpoints.broker2_endpoints.endpoints[0].id
}

provider "solacebroker" {
  alias    = "broker1"
  url      = "https://${data.solacecloud_connection_endpoint_dns_names.broker1_dns.dns_names[0].dns_name}:${data.solacecloud_connection_endpoints.broker1_endpoints.endpoints[0].ports.management_tls.port}"
  username = solacecloud_service.broker_service.message_vpn.manager_management_credential.username
  password = solacecloud_service.broker_service.message_vpn.manager_management_credential.password
}

provider "solacebroker" {
  alias    = "broker2"
  url      = "https://${data.solacecloud_connection_endpoint_dns_names.broker2_dns.dns_names[0].dns_name}:${data.solacecloud_connection_endpoints.broker2_endpoints.endpoints[0].ports.management_tls.port}"
  username = solacecloud_service.broker_service2.message_vpn.manager_management_credential.username
  password = solacecloud_service.broker_service2.message_vpn.manager_management_credential.password
}


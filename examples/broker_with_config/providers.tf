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

provider "solacebroker" {
  alias    = "broker1"
  url      = "https://${solacecloud_service.broker_service.connection_endpoints[0].hostnames[0]}:${solacecloud_service.broker_service.connection_endpoints[0].ports.management_tls.port}"
  username = solacecloud_service.broker_service.message_vpn.manager_management_credential.username
  password = solacecloud_service.broker_service.message_vpn.manager_management_credential.password
}

provider "solacebroker" {
  alias    = "broker2"
  url      = "https://${solacecloud_service.broker_service2.connection_endpoints[0].hostnames[0]}:${solacecloud_service.broker_service2.connection_endpoints[0].ports.management_tls.port}"
  username = solacecloud_service.broker_service2.message_vpn.manager_management_credential.username
  password = solacecloud_service.broker_service2.message_vpn.manager_management_credential.password
}


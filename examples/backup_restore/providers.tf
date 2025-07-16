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
  api_polling_interval = 40
}

provider "solacebroker" {
  url      = "https://${solacecloud_service.broker_service.resource_domain_name}:943"
  username = solacecloud_service.broker_service.resource_username
  password = solacecloud_service.broker_service.resource_password
}


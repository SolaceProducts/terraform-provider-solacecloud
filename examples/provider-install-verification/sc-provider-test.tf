terraform {
  required_providers {
    solacecloud = {
      source = "registry.terraform.io/solaceproducts/solacecloud"
    }
  }
}

provider "solacecloud" {
  base_url             = "https://api.solace.cloud/"
  api_polling_interval = 40
}

resource "solacecloud_service" "broker_service" {
  name             = "Azure_TF_Service"
  datacenter_id    = "eks-eu-central-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}

output "azure_service_username" {
  value = ["${solacecloud_service.broker_service.*.resource_username}"]
}

output "azure_service_password" {
  value = ["${solacecloud_service.broker_service.*.resource_password}"]
}

output "azure_service_name" {
  value = ["${solacecloud_service.broker_service.*.resource_domain_name}"]
}

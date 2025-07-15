terraform {
  required_providers {
    solacecloud = {
      source = "registry.terraform.io/solaceproducts/solacecloud"
    }
  }
}

provider "solacecloud" {
  base_url             = "https://production-api.solace.cloud"
  api_polling_interval = 30
}

import {
  id = "t3adj4smsh7"
  to = solacecloud_service.broker1
}

resource "solacecloud_service" "broker1" {
  datacenter_id    = "gke-gcp-us-central1-a"
  name             = "event-broker-service-1"
  service_class_id = "ENTERPRISE_250_STANDALONE"

}


import {
  id = "efqk2lpinva"
  to = solacecloud_service.broker2
}

resource "solacecloud_service" "broker2" {
  datacenter_id    = "gke-gcp-us-central1-a"
  name             = "event-broker-service-2"
  service_class_id = "ENTERPRISE_250_STANDALONE"
}


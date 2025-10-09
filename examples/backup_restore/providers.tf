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

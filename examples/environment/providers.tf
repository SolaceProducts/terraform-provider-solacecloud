terraform {
  required_providers {
    solacecloud = {
      source = "hashicorp.com/edu/solacecloud"
    }
  }
}

provider "solacecloud" {
  base_url             = "https://mc-rrbac-dev-api.mymaas.net/"
  api_polling_interval = 30
}

# Fetch the environment
data "solacecloud_environment" "environment" {
  name = "Test_is_suddenly_longer_than_expected_tasdfg"
}

# Create a service in the Default environment
resource "solacecloud_service" "broker_service" {
  name             = "fwanssasso-dmytro-developer-service"
  datacenter_id    = "gke-gcp-us-central1-a"
  service_class_id = "DEVELOPER"
  environment_id   = data.solacecloud_environment.environment.id
}

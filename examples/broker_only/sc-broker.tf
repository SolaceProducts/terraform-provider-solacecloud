resource "solacecloud_service" "broker_service" {
  name             = "Azure_TF_Service"
  datacenter_id    = "eks-eu-central-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}

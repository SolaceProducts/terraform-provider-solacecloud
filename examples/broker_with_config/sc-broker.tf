

resource "solacecloud_service" "broker_service" {
  name             = "JACK_TF_SERVICE"
  datacenter_id    = "eks-eu-central-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}

resource "solacecloud_service" "broker_service2" {
  name             = "COOL_SERVICE"
  datacenter_id    = "eks-eu-central-1a"
  service_class_id = "ENTERPRISE_1K_STANDALONE"
}

resource "solacebroker_msg_vpn_queue" "queue1" {
  provider = solacebroker.broker1
  queue_name = "Azure_TF_Queue1"
  msg_vpn_name  = "msgvpn-${solacecloud_service.broker_service.id}"
  ingress_enabled = true
  egress_enabled = true
  max_msg_size = 54321
  partition_count = 4
}

resource "solacebroker_msg_vpn_queue" "queue2" {
  provider = solacebroker.broker1
  queue_name = "Azure_TF_Queue2"
  msg_vpn_name  = "msgvpn-${solacecloud_service.broker_service.id}"
  ingress_enabled = true
  egress_enabled = true
  max_msg_size = 54321
  partition_count = 4
}

resource "solacebroker_msg_vpn_queue" "queue1a" {
  provider = solacebroker.broker2
  queue_name = "Azure_TF_Queue1"
  msg_vpn_name  = "msgvpn-${solacecloud_service.broker_service2.id}"
  ingress_enabled = true
  egress_enabled = true
  max_msg_size = 54321
  partition_count = 4
}

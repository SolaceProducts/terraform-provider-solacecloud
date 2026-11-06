# The event broker service must already have an LDAP authentication profile named "management".
# Shared event broker services are not supported.
variable "service_id" {
  description = "The identifier of the event broker service"
  type        = string
}

resource "solacecloud_ldap_management_access" "ldap" {
  service_id           = var.service_id
  enabled              = true
  group_attribute_name = "memberOf"

  read_only_group_names    = ["cn=broker-viewers,ou=groups,dc=example,dc=com"]
  read_write_group_names   = ["cn=broker-editors,ou=groups,dc=example,dc=com"]
  mesh_manager_group_names = ["cn=broker-managers,ou=groups,dc=example,dc=com"]
}

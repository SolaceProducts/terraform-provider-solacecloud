# Resource: solacecloud_ldap_management_access

This resource manages LDAP authentication of management users for an event broker service. It controls whether management users log in with their LDAP credentials, and which LDAP groups get which Mission Control role for the event broker service.

Each event broker service has at most one LDAP management access configuration, so the resource is identified by the event broker service ID.

**Prerequisites:**
- An existing event broker service. Shared event broker services are not supported.
- An LDAP authentication profile named `management` on that event broker service. For more information, see [Creating the Management Profile](#creating-the-management-profile).

## Example Usage

```hcl
resource "solacecloud_ldap_management_access" "ldap" {
  service_id           = solacecloud_service.broker_service.id
  enabled              = true
  group_attribute_name = "memberOf"

  read_only_group_names    = ["cn=broker-viewers,ou=groups,dc=example,dc=com"]
  read_write_group_names   = ["cn=broker-editors,ou=groups,dc=example,dc=com"]
  mesh_manager_group_names = ["cn=broker-managers,ou=groups,dc=example,dc=com"]
}
```

## Argument Reference

### Required Arguments

* `service_id` - (Required) The identifier of the event broker service. Changing it replaces the resource.
* `enabled` - (Required) Indicates whether management users are authenticated against the `management` LDAP authentication profile.
* `group_attribute_name` - (Required) The name of the LDAP attribute that carries the group membership of a management user, for example `memberOf`.

### Optional Arguments

* `read_only_group_names` - (Optional) The LDAP groups whose members get Mission Control Viewer access to the event broker service.
* `read_write_group_names` - (Optional) The LDAP groups whose members get Mission Control Editor access to the event broker service.
* `mesh_manager_group_names` - (Optional) The LDAP groups whose members get Mission Control Manager access to the event broker service. The event broker service must support the Mission Control Manager role.

At least one of the three group lists must contain a group. An omitted list is empty, so removing a list from your configuration removes access for all of its groups.

Group names must not be blank, must not contain `|`, and must not repeat within a list. Each list must stay within 2,048 characters when its names are joined. The order of names in a list does not matter.

## Attribute Reference

* `id` - The unique identifier of the event broker service.
* `management_profile_name` - The name of the LDAP authentication profile that management users are authenticated against. It is `management` for configurations made with this resource. For more information, see [Event Broker Services Configured Through an Earlier API](#event-broker-services-configured-through-an-earlier-api).

## Creating the Management Profile

This resource does not create the LDAP authentication profile. Create a profile named `management` in the Cloud Console, or with the Mission Control API:

```bash
curl -X POST "https://production-api.solace.cloud/api/v2/missionControl/eventBrokerServices/${SERVICE_ID}/ldapAuthenticationProfiles" \
  -H "Authorization: Bearer ${SOLACECLOUD_API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"name": "management", "enabled": true, "ldapServerOne": "ldaps://ldap.example.com:636", "searchBaseDn": "dc=example,dc=com", "startTls": false}'
```

Without the profile, creating this resource fails with an error saying the event broker service has no `management` LDAP authentication profile.

## Disabling or Destroying

Setting `enabled = false` and destroying the resource both switch management authentication on the event broker service from LDAP to the event broker's internal authentication. After either change, LDAP users can no longer log in to Broker Manager.

The two differ in what happens to the configuration:

* If you set `enabled = false`, the configuration is kept, including the group lists. To turn LDAP authentication back on, set `enabled = true`.
* If you destroy the resource, the configuration is removed.

## Update Behavior

Changes to `enabled`, `group_attribute_name` and the group lists are applied in place. The provider sends only the arguments that changed and waits for the update to finish. Changing `service_id` replaces the resource.

An event broker service runs one operation at a time. If the same `terraform apply` also changes the event broker service or another resource on it, the apply can fail with "Another operation is already in progress." Add `depends_on` so the changes run one after another, or apply again.

## Delete Behavior

Deleting the resource removes the LDAP management access configuration from the event broker service and waits for the deletion to finish. If the configuration was already removed outside Terraform, the deletion succeeds.

## Import

You can import LDAP management access using the event broker service ID.

**Using Terraform 1.5+ import blocks:**

```hcl
import {
  to = solacecloud_ldap_management_access.ldap
  id = "myserviceid"
}

resource "solacecloud_ldap_management_access" "ldap" {
  service_id            = "myserviceid"
  enabled               = true
  group_attribute_name  = "memberOf"
  read_only_group_names = ["cn=broker-viewers,ou=groups,dc=example,dc=com"]
}
```

**Using the command line (for one-off imports):**

```bash
terraform import solacecloud_ldap_management_access.ldap myserviceid
```

## Troubleshooting

### "LDAP management access is already configured"

The event broker service already has a configuration that Terraform does not manage, for example, one made in the Cloud Console. Import it instead of creating it.

This error also happens if the Terraform process was stopped during a create, after the create request was accepted. The configuration exists on the event broker service but not in your Terraform state. Import it, then apply again.

### Event Broker Services Configured Through an Earlier API

An event broker service configured through an earlier version of the API can report a `management_profile_name` other than `management`. This resource can read and import that configuration, but it cannot change or delete it, and `terraform apply` or `terraform destroy` fails with an error naming the profile. To stop managing the configuration with Terraform, remove it from your state:

```bash
terraform state rm solacecloud_ldap_management_access.ldap
```

This directory contains the basics to deploy a new broker Solace Cloud and then configure with the configuration
coming from another broker.

The first step would be to pull the configuration from a broker:
```bash
export SEMP_URL=https://<Solace cloud broker hostname>:943
export OUTPUT_MSG_VPN_NAME=<Solace Cloud broker VPN name>
export SOLACEBROKER_USERNAME=<SEMP username from solace cloud>
export SOLACEBROKER_PASSWORD=<SEMP password from solace cloud>

terraform-provider-solacebroker generate --url=${SEMP_URL} solacebroker_msg_vpn.exported_vpn ${OUTPUT_MSG_VPN_NAME} exported-vpn.tf
```

The above steps will create the `exported-vpn.tf` file.  We need to patch the generated exported-vpn.tf file so that
it can be deployed on top of the broker defined in this directory:

# Remove the terraform block, the three broker_* variables blocks and then the provider "solacebroker" block.
## They should be the first five blocks.  They need to be removed because our Solace Cloud provider will be defining these blocks instead.
# Remove the solacebroker_msg_vpn block that follows the block deleted in step 1.
## We do not want to backup the VPNs config using the broker dSemp provider.  This is because this config is handled by Solace Cloud, and Solace Cloud SEMP users doesn’t have access to this config.
# Replace all occurrences of the solacebroker_msg_vpn.exported_vpn.msg_vpn_name string by "msgvpn-${solacecloudse_scservice.broker_service.resource_id}" (Including the ").
## This is needed because we want to use the VPN name of the target broker to which we restore.
# Delete all solacebroker_msg_vpn_client_profile blocks.  This is similar to #2, where we cannot import configuration that’s defined by Solace Cloud.


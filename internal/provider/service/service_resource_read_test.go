package service

import (
	"context"
	"terraform-provider-solacecloud/internal"
	"terraform-provider-solacecloud/internal/provider/apiclient"
	"terraform-provider-solacecloud/missioncontrol"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jarcoal/httpmock"
)

// Service GET response for a service created with DMR off: the broker object
// has no "cluster" attribute at all.
const noClusterServiceResponse = `{
    "data": {
        "id": "svc-dmr-off",
        "type": "service",
        "name": "dmr-off-service",
        "ownedBy": "test-user",
        "datacenterId": "eks-us-east-1",
        "serviceClassId": "ENTERPRISE_250_STANDALONE",
        "eventBrokerServiceVersion": "10.25.0",
        "msgVpnName": "msgvpn-test",
        "broker": {
            "version": "10.25.0",
            "maxSpoolUsage": 20,
            "redundancyGroupSslEnabled": false,
            "managementReadOnlyLoginCredential": {
                "username": "msgvpn-test-view",
                "password": "view-password"
            },
            "msgVpns": [
                {
                    "msgVpnName": "msgvpn-test",
                    "authenticationBasicEnabled": true,
                    "authenticationBasicType": "INTERNAL",
                    "authenticationClientCertEnabled": false,
                    "authenticationClientCertValidateDateEnabled": false,
                    "managementAdminLoginCredential": {
                        "username": "msgvpn-test-admin",
                        "password": "admin-password"
                    },
                    "missionControlManagerLoginCredential": {
                        "username": "mission-control-manager",
                        "password": "manager-password"
                    },
                    "serviceLoginCredential": {
                        "username": "solace-cloud-client",
                        "password": "client-password"
                    },
                    "maxConnectionCount": 1000,
                    "maxEgressFlowCount": 1000,
                    "maxEndpointCount": 1000,
                    "maxIngressFlowCount": 1000,
                    "maxMsgSpoolUsage": 20000,
                    "maxSubscriptionCount": 100000,
                    "maxTransactedSessionCount": 1000,
                    "maxTransactionCount": 5000
                }
            ]
        }
    }
}`

// TestReadDataInternalWithoutCluster verifies that reading a service created
// with DMR off (no "cluster" object in the broker response) does not panic and
// maps cluster_name and dmr_cluster to null values in state.
func TestReadDataInternalWithoutCluster(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseUrl := "https://api.solace.test"
	httpmock.RegisterResponder(
		"GET",
		`=~^https://api\.solace\.test/api/v2/missionControl/eventBrokerServices/svc-dmr-off`,
		internal.JsonResponder(200, noClusterServiceResponse))

	mcClient, err := missioncontrol.NewClientWithResponses(baseUrl)
	if err != nil {
		t.Fatalf("failed to create mission control client: %v", err)
	}

	r := &ServiceResource{
		APIClient:          apiclient.NewRetryableClient(mcClient, 1, 0),
		APIPollingInterval: 1,
	}

	currentState := &ServiceResourceModel{
		Id: types.StringValue("svc-dmr-off"),
	}

	diags := r.readDataInternal(context.Background(), currentState, nil)
	if diags.HasError() {
		t.Fatalf("readDataInternal returned diagnostics errors: %v", diags)
	}

	if !currentState.ClusterName.IsNull() {
		t.Errorf("expected cluster_name to be null for a DMR-off service, got: %s", currentState.ClusterName)
	}
	if !currentState.DmrClusterInfo.IsNull() {
		t.Errorf("expected dmr_cluster to be null for a DMR-off service, got: %s", currentState.DmrClusterInfo)
	}
	if !currentState.CustomRouterName.IsNull() {
		t.Errorf("expected custom_router_name to be null for a DMR-off service, got: %s", currentState.CustomRouterName)
	}
	if currentState.Name.ValueString() != "dmr-off-service" {
		t.Errorf("expected name to be dmr-off-service, got: %s", currentState.Name)
	}
}

// TestReadDataInternalWithoutClusterPreservesPlannedCustomRouterName verifies
// that on Create/Update (non-nil initialPlan) of a DMR-off service, a
// custom_router_name from the plan is carried into state instead of being
// dropped, which would make the apply result inconsistent with the plan.
func TestReadDataInternalWithoutClusterPreservesPlannedCustomRouterName(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseUrl := "https://api.solace.test"
	httpmock.RegisterResponder(
		"GET",
		`=~^https://api\.solace\.test/api/v2/missionControl/eventBrokerServices/svc-dmr-off`,
		internal.JsonResponder(200, noClusterServiceResponse))

	mcClient, err := missioncontrol.NewClientWithResponses(baseUrl)
	if err != nil {
		t.Fatalf("failed to create mission control client: %v", err)
	}

	r := &ServiceResource{
		APIClient:          apiclient.NewRetryableClient(mcClient, 1, 0),
		APIPollingInterval: 1,
	}

	currentState := &ServiceResourceModel{
		Id: types.StringValue("svc-dmr-off"),
	}
	initialPlan := &ServiceResourceModel{
		CustomRouterName: types.StringValue("myrouter"),
	}

	diags := r.readDataInternal(context.Background(), currentState, initialPlan)
	if diags.HasError() {
		t.Fatalf("readDataInternal returned diagnostics errors: %v", diags)
	}

	if !currentState.ClusterName.IsNull() {
		t.Errorf("expected cluster_name to be null for a DMR-off service, got: %s", currentState.ClusterName)
	}
	if !currentState.DmrClusterInfo.IsNull() {
		t.Errorf("expected dmr_cluster to be null for a DMR-off service, got: %s", currentState.DmrClusterInfo)
	}
	if currentState.CustomRouterName.ValueString() != "myrouter" {
		t.Errorf("expected custom_router_name from the plan to be preserved, got: %s", currentState.CustomRouterName)
	}
}

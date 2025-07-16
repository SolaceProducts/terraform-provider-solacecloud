package internal

import (
	"fmt"
	"time"

	"github.com/jarcoal/httpmock"
)

// Enhanced mock scenarios for comprehensive acceptance testing

// SetupCreationFailureMocks sets up mocks for testing creation failure scenarios
func (provider *TestInstance) SetupCreationFailureMocks() {
	httpmock.Activate()

	// Mock API validation error
	httpmock.RegisterResponder(
		"POST",
		provider.baseUrl+"/api/v2/missionControl/eventBrokerServices",
		JsonResponder(400, `{
			"error": {
				"code": "INVALID_REQUEST",
				"message": "Invalid service class specified",
				"details": "Service class 'INVALID_CLASS' is not supported"
			}
		}`))
}

// SetupAsyncCreationMocks sets up mocks for testing async creation patterns
func (provider *TestInstance) SetupAsyncCreationMocks(params ConfigurableParams) {
	httpmock.Activate()

	// Initial creation request returns operation ID
	httpmock.RegisterResponder(
		"POST",
		provider.baseUrl+"/api/v2/missionControl/eventBrokerServices",
		JsonResponder(202, `{
			"data": {
				"id": "async-op-123",
				"type": "operation",
				"operationType": "createService",
				"createdBy": "test-user",
				"createdTime": "`+time.Now().Format(time.RFC3339)+`",
				"resourceId": "pending-service-id",
				"resourceType": "service",
				"status": "PENDING",
				"error": null
			}
		}`))

	// First GET request shows service in progress
	httpmock.RegisterResponder(
		"GET",
		provider.baseUrl+"/api/v2/missionControl/eventBrokerServices/pending-service-id",
		httpmock.NewStringResponder(200, fmt.Sprintf(`{
			"data": {
				"id": "pending-service-id",
				"type": "service",
				"name": "%s",
				"serviceClassId": "%s",
				"creationState": "IN_PROGRESS",
				"adminState": "PENDING"
			}
		}`, params.ServiceName, params.ServiceClass)).Times(1))

	// Subsequent GET requests show service completed (with expand parameters)
	httpmock.RegisterResponder(
		"GET",
		provider.baseUrl+"/api/v2/missionControl/eventBrokerServices/pending-service-id",
		JsonResponder(200, createAsyncServiceResponse(params)))

	// Also handle GET requests with expand parameters (exact match for readDataInternal)
	httpmock.RegisterResponder(
		"GET",
		provider.baseUrl+"/api/v2/missionControl/eventBrokerServices/pending-service-id?expand=broker%2CserviceConnectionEndpoints%2CmessageSpoolDetails",
		JsonResponder(200, createAsyncServiceResponse(params)))

	// Mock DELETE for cleanup
	httpmock.RegisterResponder(
		"DELETE",
		provider.baseUrl+"/api/v2/missionControl/eventBrokerServices/pending-service-id",
		JsonResponder(202, `{"data": {"id": "delete-op-123", "status": "PENDING"}}`))
}

// SetupRateLimitMocks sets up mocks for testing rate limiting scenarios
func (provider *TestInstance) SetupRateLimitMocks() {
	httpmock.Activate()

	// First request hits rate limit
	httpmock.RegisterResponder(
		"POST",
		provider.baseUrl+"/api/v2/missionControl/eventBrokerServices",
		httpmock.NewStringResponder(429, `{
			"error": {
				"code": "RATE_LIMIT_EXCEEDED",
				"message": "Too many requests"
			}
		}`).Times(1))

	// Subsequent request succeeds
	httpmock.RegisterResponder(
		"POST",
		provider.baseUrl+"/api/v2/missionControl/eventBrokerServices",
		JsonResponder(202, `{
			"data": {
				"id": "rate-limit-op-456",
				"type": "operation",
				"operationType": "createService",
				"status": "PENDING",
				"resourceId": "6q1p55o6ovr"
			}
		}`))
}

// SetupConflictMocks sets up mocks for testing resource conflict scenarios
func (provider *TestInstance) SetupConflictMocks() {
	httpmock.Activate()

	httpmock.RegisterResponder(
		"POST",
		provider.baseUrl+"/api/v2/missionControl/eventBrokerServices",
		JsonResponder(409, `{
			"error": {
				"code": "RESOURCE_CONFLICT",
				"message": "Service with this name already exists",
				"details": "A service named 'duplicate-service' already exists in this environment"
			}
		}`))
}

// SetupPartialCreationMocks sets up mocks for testing partial creation scenarios
func (provider *TestInstance) SetupPartialCreationMocks(params ConfigurableParams) {
	httpmock.Activate()

	// Creation succeeds but service is in failed state
	httpmock.RegisterResponder(
		"POST",
		provider.baseUrl+"/api/v2/missionControl/eventBrokerServices",
		JsonResponder(202, `{
			"data": {
				"id": "partial-op-789",
				"type": "operation",
				"operationType": "createService",
				"status": "PENDING",
				"resourceId": "failed-service-id"
			}
		}`))

	// Service exists but in failed state
	httpmock.RegisterResponder(
		"GET",
		provider.baseUrl+"/api/v2/missionControl/eventBrokerServices/failed-service-id",
		JsonResponder(200, createFailedServiceResponse(params)))
}

func createFailedServiceResponse(params ConfigurableParams) string {
	return fmt.Sprintf(`{
		"data": {
			"id": "failed-service-id",
			"type": "service",
			"name": "%s",
			"serviceClassId": "%s",
			"creationState": "FAILED",
			"adminState": "STOP",
			"locked": true,
			"error": {
				"code": "INFRASTRUCTURE_ERROR",
				"message": "Failed to provision infrastructure"
			}
		}
	}`, params.ServiceName, params.ServiceClass)
}

func createAsyncServiceResponse(params ConfigurableParams) string {
	return fmt.Sprintf(`{
		"data": {
			"id": "pending-service-id",
			"type": "service",
			"name": "%s",
			"eventBrokerServiceVersion": "10.10.1.112-3",
			"createdTime": "2025-02-19T01:18:36Z",
			"ownedBy": "test-user",
			"infrastructureId": "62xaikge1vm",
			"datacenterId": "eks-us-east-1",
			"serviceClassId": "%s",
			"adminState": "START",
			"creationState": "COMPLETED",
			"locked": false,
			"allowedActions": [
				"update",
				"delete",
				"assign",
				"get",
				"configure",
				"broker_update"
			],
			"messageSpoolDetails": {
				"expandedGbBilled": 0,
				"defaultGbSize": 200,
				"totalGbSize": 200
			},
			"environmentId": "test-env-id",
			"msgVpnName": "msgvpn-pending-service-id",
			"defaultManagementHostname": "mr-connection-pending-service-id.messaging.solace.cloud",
			"serviceConnectionEndpoints": [
				{
					"id": "pending-service-id",
					"type": "serviceConnectionEndpoint",
					"name": "Default Public",
					"description": "",
					"accessType": "PUBLIC",
					"k8sServiceType": "LOADBALANCER",
					"k8sServiceId": "kilo-sa-production-pending-service-id-solace",
					"hostNames": [
						"mr-connection-pending-service-id.messaging.solace.cloud"
					],
					"ports": [
						{
							"protocol": "serviceWebPlainTextListenPort",
							"port": 0
						},
						{
							"protocol": "serviceManagementTlsListenPort",
							"port": 943
						},
						{
							"protocol": "serviceRestIncomingTlsListenPort",
							"port": 9443
						},
						{
							"protocol": "serviceAmqpPlainTextListenPort",
							"port": 0
						},
						{
							"protocol": "serviceMqttWebSocketListenPort",
							"port": 0
						},
						{
							"protocol": "serviceRestIncomingPlainTextListenPort",
							"port": 0
						},
						{
							"protocol": "serviceWebTlsListenPort",
							"port": 443
						},
						{
							"protocol": "serviceSmfCompressedListenPort",
							"port": 0
						},
						{
							"protocol": "serviceMqttPlainTextListenPort",
							"port": 0
						},
						{
							"protocol": "serviceSmfPlainTextListenPort",
							"port": 0
						},
						{
							"protocol": "serviceAmqpTlsListenPort",
							"port": 5671
						},
						{
							"protocol": "serviceMqttTlsListenPort",
							"port": 8883
						},
						{
							"protocol": "serviceSmfTlsListenPort",
							"port": 55443
						},
						{
							"protocol": "serviceMqttTlsWebSocketListenPort",
							"port": 8443
						},
						{
							"protocol": "managementSshTlsListenPort",
							"port": 22
						}
					]
				}
			],
			"broker": {
				"version": "10.10.1.112",
				"versionFamily": "10.10",
				"maxSpoolUsage": 200,
				"diskSize": 260,
				"redundancyGroupSslEnabled": false,
				"configSyncSslEnabled": true,
				"monitoringMode": "BASIC",
				"tlsStandardDomainCertificateAuthoritiesEnabled": true,
				"cluster": {
					"name": "test-cluster",
					"password": "dmr-password",
					"remoteAddress": "mr-connection-pending-service-id.messaging.solace.cloud",
					"primaryRouterName": "pending-service-id-primary",
					"supportedAuthenticationMode": [
						"Basic"
					]
				},
				"managementReadOnlyLoginCredential": {
					"username": "msgvpn-pending-service-id-view",
					"password": "viewer-password",
					"token": "YWJj.eyJhY2Nlc3NfdG9rZW4iOiAibXNndnBuLXBlbmRpbmctc2VydmljZS1pZC12aWV3OnZpZXdlci1wYXNzd29yZCJ9.eHl6"
				},
				"msgVpns": [
					{
						"msgVpnName": "msgvpn-pending-service-id",
						"authenticationBasicEnabled": true,
						"authenticationBasicType": "INTERNAL",
						"authenticationClientCertEnabled": false,
						"authenticationClientCertValidateDateEnabled": false,
						"clientProfiles": [
							{
								"name": "default"
							}
						],
						"enabled": true,
						"eventLargeMsgThreshold": 11,
						"managementAdminLoginCredential": {
							"username": "msgvpn-pending-service-id-admin",
							"password": "editor-password",
							"token": "YWJj.eyJhY2Nlc3NfdG9rZW4iOiAibXNndnBuLXBlbmRpbmctc2VydmljZS1pZC1hZG1pbjplZGl0b3ItcGFzc3dvcmQifQ%%3D%%3D.eHl6"
						},
						"missionControlManagerLoginCredential": {
							"username": "mission-control-manager",
							"password": "manager-password",
							"token": "YWJj.eyJhY2Nlc3NfdG9rZW4iOiAibWlzc2lvbi1jb250cm9sLW1hbmFnZXI6bWFuYWdlci1wYXNzd29yZCJ9.eHl6"
						},
						"serviceLoginCredential": {
							"username": "solace-cloud-client",
							"password": "default-password"
						},
						"maxConnectionCount": 1000,
						"maxEgressFlowCount": 1000,
						"maxEndpointCount": 1000,
						"maxIngressFlowCount": 1000,
						"maxMsgSpoolUsage": 200000,
						"maxSubscriptionCount": 100000,
						"maxTransactedSessionCount": 1000,
						"maxTransactionCount": 5000,
						"sempOverMessageBus": {
							"sempOverMsgBusEnabled": false,
							"sempAccessToShowCmdsEnabled": false,
							"sempAccessToAdminCmdsEnabled": false,
							"sempAccessToClientAdminCmdsEnabled": false,
							"sempAccessToCacheCmdsEnabled": false
						},
						"subDomainName": "mr-connection-pending-service-id.messaging.solace.cloud",
						"truststoreUri": "https://cacerts.digicert.com/DigiCertGlobalRootCA.crt.pem"
					}
				]
			}
		}
	}`, params.ServiceName, params.ServiceClass)
}

// CreateCompleteServiceResponse creates a complete service response with custom ID and name
func CreateCompleteServiceResponse(serviceId, serviceName, serviceClass string) string {
	return fmt.Sprintf(`{
		"data": {
			"id": "%s",
			"type": "service",
			"name": "%s",
			"serviceClassId": "%s",
			"datacenterId": "eks-us-east-1",
			"creationState": "COMPLETED",
			"adminState": "START",
			"locked": false,
			"eventBrokerServiceVersion": "10.10.1.112-3",
			"msgVpnName": "msgvpn-%s",
			"environmentId": "test-env-id",
			"ownedBy": "test-user",
			"clusterName": "test-cluster",
			"allowedActions": ["update", "delete", "get"],
			"messageSpoolDetails": {
				"totalGbSize": 200
			},
			"serviceConnectionEndpoints": [{
				"id": "endpoint-1",
				"name": "Public",
				"description": "Public endpoint",
				"accessType": "public",
				"hostnames": ["test.messaging.solace.cloud"],
				"ports": {
					"smfTls": {"port": 55443},
					"webTls": {"port": 443},
					"mqttTls": {"port": 8883},
					"mqttWebsocketTls": {"port": 443},
					"restIncomingTls": {"port": 9443},
					"amqpTls": {"port": 5671},
					"managementTls": {"port": 943},
					"sshTls": {"port": 2222}
				}
			}],
			"broker": {
				"maxSpoolUsage": 200,
				"redundancyGroupSslEnabled": false,
				"cluster": {
					"name": "test-cluster",
					"primaryRouterName": "primary-router",
					"remoteAddress": "test.messaging.solace.cloud:55003",
					"password": "dmr-password",
					"supportedAuthenticationMode": ["basic"]
				},
				"managementReadOnlyLoginCredential": {
					"username": "viewer",
					"password": "viewer-password"
				},
				"msgVpns": [{
					"msgVpnName": "msgvpn-%s",
					"maxConnectionCount": 1000,
					"maxEndpointCount": 1000,
					"maxIngressFlowCount": 1000,
					"maxEgressFlowCount": 1000,
					"maxMsgSpoolUsage": 200,
					"maxSubscriptionCount": 5000000,
					"maxTransactionCount": 500,
					"maxTransactedSessionCount": 500,
					"authenticationBasicEnabled": true,
					"authenticationBasicType": "internal",
					"authenticationClientCertEnabled": false,
					"authenticationClientCertValidateDateEnabled": true,
					"truststoreUri": "",
					"serviceLoginCredential": {
						"username": "default",
						"password": "default-password"
					},
					"managementAdminLoginCredential": {
						"username": "editor",
						"password": "editor-password"
					},
					"missionControlManagerLoginCredential": {
						"username": "manager",
						"password": "manager-password"
					}
				}]
			}
		}
	}`, serviceId, serviceName, serviceClass, serviceId, serviceId)
}

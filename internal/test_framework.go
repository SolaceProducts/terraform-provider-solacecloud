package internal

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"terraform-provider-solacecloud/missioncontrol"

	"github.com/jarcoal/httpmock"
	"github.com/labstack/gommon/random"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
)

// This is a test framework for the Solace Cloud Terraform provider.
// All shared test code should be saved in this file.

type TestInstance struct {
	baseUrl   string
	mockedApi bool
	client    *missioncontrol.ClientWithResponses
}

// constructor that initializes base_url from enviornment variable
func NewTestInstance() *TestInstance {
	return &TestInstance{
		baseUrl:   "https://api.solace.com",
		mockedApi: false,
	}
}

func (provider *TestInstance) IsMocked() bool {
	return provider.mockedApi
}

func (provider *TestInstance) GetBaseURL() string {
	return provider.baseUrl
}
func (provider *TestInstance) GetClient() *missioncontrol.ClientWithResponses {
	return provider.client
}
func (provider *TestInstance) Init(params ConfigurableParams) {
	baseUrl := os.Getenv("SOLACE_BASE_URL")
	provider.baseUrl = baseUrl
	if baseUrl == "" {
		provider.baseUrl = "http://" + random.String(9) + ".com"
		provider.mockedApi = true
		httpmock.Activate()
		provider.SetupDefaultMocks(params)
		return
	} else {
		provider.setupHttpClient(context.Background())

	}

}

func (provider *TestInstance) setupHttpClient(ctx context.Context) {
	baseUrl := provider.baseUrl
	apiToken := os.Getenv("SOLACECLOUD_API_TOKEN")
	tokenAuth, _ := securityprovider.NewSecurityProviderBearerToken(apiToken)
	apiClient, err := missioncontrol.NewClientWithResponses(
		baseUrl,
		missioncontrol.WithRequestEditorFn(tokenAuth.Intercept),
		// Add a Request Editor to set the Content-Type to JSON for all requests
		func(c *missioncontrol.Client) error {
			c.RequestEditors = append(c.RequestEditors, func(ctx context.Context, req *http.Request) error {
				req.Header.Set("Accept", "application/json")
				return nil
			})
			return nil
		})
	if err != nil {
		// Handle the error appropriately - could log, panic, or return depending on requirements
		panic("Failed to create API client: " + err.Error())
	}
	provider.client = apiClient
}

func JsonResponder(status int, body string) httpmock.Responder {
	//json header
	return httpmock.NewStringResponder(status, body).HeaderAdd(http.Header{
		"Content-Type": []string{"application/json"},
	})
}

type ConfigurableParams struct {
	ServiceName      string
	ServiceClass     string
	ServiceId        string
	Locked           bool
	MaxSpoolUsage    int
	OwnerId          string
	CustomRouterName string
}

func (provider *TestInstance) SetupDefaultMocks(params ConfigurableParams) {
	if params.ServiceId == "" {
		params.ServiceId = "6q1p55o6ovr"
	}
	if params.MaxSpoolUsage == 0 {
		params.MaxSpoolUsage = 200
	}

	if params.OwnerId == "" {
		params.OwnerId = "67tr8tkuel"
	}

	httpmock.RegisterResponder(
		"POST",
		provider.baseUrl+"/api/v2/missionControl/eventBrokerServices",
		JsonResponder(202, `{
    "data": {
        "id": "44x9yacy20i",
        "type": "operation",
        "operationType": "createService",
        "createdBy": "67tr8tkuel",
        "createdTime": "2025-02-19T01:33:04Z",
        "completedTime": "2025-02-19T01:33:04Z",
        "resourceId": "`+params.ServiceId+`",
        "resourceType": "service",
        "status": "PENDING",
        "error": null
    }
}`))

	httpmock.RegisterResponder("GET", provider.baseUrl+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId,
		JsonResponder(200, CreateGetServiceResponse(params)))

	httpmock.RegisterResponder("GET", provider.baseUrl+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"?expand=broker,serviceConnectionEndpoints,allowedActions,messageSpoolDetails",
		JsonResponder(200, CreateGetServiceResponse(params)))

	httpmock.RegisterResponder("DELETE", provider.baseUrl+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId,
		JsonResponder(202, `
{
    "data": {
        "id": "797mg2xoffj",
        "type": "operation",
        "operationType": "deleteService",
        "createdBy": "67tr8tkuel",
        "createdTime": "2025-02-19T02:19:16Z",
        "completedTime": "2025-02-19T02:19:16Z",
        "resourceId": "`+params.ServiceId+`",
        "resourceType": "service",
        "status": "PENDING",
        "error": null
    }
}`))

}

func CreateGetServiceResponse(params ConfigurableParams) string {
	return `
{
    "data": {
        "id": "` + params.ServiceId + `",
        "type": "service",
        "name": "` + params.ServiceName + `",
        "eventBrokerServiceVersion": "10.10.1.112-3",
        "createdTime": "2025-02-19T01:18:36Z",
        "ownedBy": ` + strconv.Quote(params.OwnerId) + `,
        "infrastructureId": "62xaikge1vm",
        "datacenterId": "eks-us-east-1",
        "serviceClassId": "` + params.ServiceClass + `",
        "adminState": "START",
        "creationState": "COMPLETED",
        "locked": ` + strconv.FormatBool(params.Locked) + `,
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
            "defaultGbSize": ` + strconv.Itoa(params.MaxSpoolUsage) + `,
            "totalGbSize": ` + strconv.Itoa(params.MaxSpoolUsage) + `
        },
        "environmentId": "5vfimhe1w8v",
        "msgVpnName": "msgvpn-6q1p55o6ovr",
        "defaultManagementHostname": "mr-connection-80dx8er674q.messaging.solace.cloud",
        "serviceConnectionEndpoints": [
            {
                "id": "80dx8er674q",
                "type": "serviceConnectionEndpoint",
                "name": "Default Public",
                "description": "",
                "accessType": "PUBLIC",
                "k8sServiceType": "LOADBALANCER",
                "k8sServiceId": "kilo-sa-production-80dx8er674q-solace",
                "hostNames": [
                    "mr-connection-80dx8er674q.messaging.solace.cloud"
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
            "maxSpoolUsage": ` + strconv.Itoa(params.MaxSpoolUsage) + `,
            "diskSize": 260,
            "redundancyGroupSslEnabled": true,
            "configSyncSslEnabled": true,
            "monitoringMode": "BASIC",
            "tlsStandardDomainCertificateAuthoritiesEnabled": true,
            "cluster": {
                "name": "cluster-eks-eu-central-1a-3-62xaikge1vm",
                "password": "3hg48osubca8t5blpqjcge0ju",
                "remoteAddress": "mr-connection-80dx8er674q.messaging.solace.cloud",
                "primaryRouterName": "` + determineRouterName(params.CustomRouterName) + `",
                "supportedAuthenticationMode": [
                    "Basic"
                ]
            },
            "managementReadOnlyLoginCredential": {
                "username": "msgvpn-6q1p55o6ovr-view",
                "password": "5k02537vrlu59fq7sc4epur7ug",
                "token": "YWJj.eyJhY2Nlc3NfdG9rZW4iOiAibXNndnBuLTZxMXA1NW82b3ZyLXZpZXc6NWswMjUzN3ZybHU1OWZxN3NjNGVwdXI3dWcifQ%3D%3D.eHl6"
            },
            "msgVpns": [
                {
                    "msgVpnName": "msgvpn-6q1p55o6ovr",
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
                        "username": "msgvpn-6q1p55o6ovr-admin",
                        "password": "1uc3kqk37ll1vd0k089r7p8cb8",
                        "token": "YWJj.eyJhY2Nlc3NfdG9rZW4iOiAibXNndnBuLTZxMXA1NW82b3ZyLWFkbWluOjF1YzNrcWszN2xsMXZkMGswODlyN3A4Y2I4In0%3D.eHl6"
                    },
                    "missionControlManagerLoginCredential": {
                        "username": "mission-control-manager",
                        "password": "g0mcefpt10sn7pq4sfaqo4v6qs",
                        "token": "YWJj.eyJhY2Nlc3NfdG9rZW4iOiAibWlzc2lvbi1jb250cm9sLW1hbmFnZXI6ZzBtY2VmcHQxMHNuN3BxNHNmYXFvNHY2cXMifQ%3D%3D.eHl6"
                    },
                    "serviceLoginCredential": {
                        "username": "solace-cloud-client",
                        "password": "sq23tgo3tsiq15ui7eu3u0epvc"
                    },
                    "maxConnectionCount": 1000,
                    "maxEgressFlowCount": 1000,
                    "maxEndpointCount": 1000,
                    "maxIngressFlowCount": 1000,
                    "maxMsgSpoolUsage": ` + strconv.Itoa(params.MaxSpoolUsage*1000) + `,
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
                    "subDomainName": "mr-connection-80dx8er674q.messaging.solace.cloud",
                    "truststoreUri": "https://cacerts.digicert.com/DigiCertGlobalRootCA.crt.pem"
                }
            ]
        }
    }
}`

}

func determineRouterName(s string) string {
	if s == "" {
		return "6q1p55o6ovrprimary"
	} else {
		return s + "primarycn"

	}
}

func (provider *TestInstance) GetBaseHcl() string {
	if provider.mockedApi {
		return `
provider "solacecloud" {
  base_url             = "` + provider.baseUrl + `"
  api_polling_interval = 1
  api_token            = "mocked_api_token"
}
`
	}
	return `
provider "solacecloud" {
  base_url             = "` + provider.baseUrl + `"
  api_polling_interval = 1
}
`
}

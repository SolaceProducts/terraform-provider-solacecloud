package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
)

// decodeServiceResponse parses a mocked GET service body and returns its "data" object.
func decodeServiceResponse(t *testing.T, body string) map[string]interface{} {
	t.Helper()
	var parsed struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("mocked service response is not valid JSON: %v", err)
	}
	return parsed.Data
}

func TestCreateServiceOperationResponse(t *testing.T) {
	cases := []struct {
		name           string
		params         ConfigurableParams
		wantResourceId string
	}{
		{name: "defaults the service id", params: ConfigurableParams{}, wantResourceId: "6q1p55o6ovr"},
		{name: "uses the configured service id", params: ConfigurableParams{ServiceId: "svc-custom"}, wantResourceId: "svc-custom"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := decodeServiceResponse(t, CreateServiceOperationResponse(tc.params))
			if data["resourceId"] != tc.wantResourceId {
				t.Errorf("expected resourceId %q, got %v", tc.wantResourceId, data["resourceId"])
			}
			if data["operationType"] != "createService" {
				t.Errorf("expected operationType createService, got %v", data["operationType"])
			}
		})
	}
}

func TestCreateGetServiceResponseDmrShape(t *testing.T) {
	cases := []struct {
		name            string
		params          ConfigurableParams
		wantCluster     bool
		wantClusterName string
	}{
		{name: "DMR on by default with the generated cluster name", params: ConfigurableParams{},
			wantCluster: true, wantClusterName: "cluster-eks-eu-central-1a-3-62xaikge1vm"},
		{name: "DMR on with a configured cluster name", params: ConfigurableParams{ClusterName: "named-cluster"},
			wantCluster: true, wantClusterName: "named-cluster"},
		{name: "DMR off has no cluster object", params: ConfigurableParams{DmrOff: true},
			wantCluster: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			params := tc.params
			params.ServiceId = "svc-shape"
			params.ServiceName = "shape-test"
			params.ServiceClass = "ENTERPRISE_1K_STANDALONE"

			data := decodeServiceResponse(t, CreateGetServiceResponse(params))
			broker, ok := data["broker"].(map[string]interface{})
			if !ok {
				t.Fatalf("expected a broker object, got %v", data["broker"])
			}

			cluster, hasCluster := broker["cluster"]
			if hasCluster != tc.wantCluster {
				t.Fatalf("expected cluster present=%t, got %v", tc.wantCluster, cluster)
			}
			if tc.wantCluster {
				if name := cluster.(map[string]interface{})["name"]; name != tc.wantClusterName {
					t.Errorf("expected cluster name %q, got %v", tc.wantClusterName, name)
				}
			}

			msgVpns := broker["msgVpns"].([]interface{})
			if dmrEnabled := msgVpns[0].(map[string]interface{})["dmrEnabled"]; dmrEnabled != tc.wantCluster {
				t.Errorf("expected msgVpns[0].dmrEnabled=%t, got %v", tc.wantCluster, dmrEnabled)
			}
		})
	}
}

// TestSetupDmrClusterMocksEnableFlipsTheServiceShape verifies the DMR mock helper: an enable
// request returns an operation and makes a later read report a service that has a cluster.
func TestSetupDmrClusterMocksEnableFlipsTheServiceShape(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	instance := &TestInstance{baseUrl: "https://api.solace.test", mockedApi: true}
	params := ConfigurableParams{
		ServiceId:    "svc-mock-dmr",
		ServiceName:  "mock-dmr",
		ServiceClass: "ENTERPRISE_1K_STANDALONE",
		DmrOff:       true,
	}
	instance.SetupDefaultMocks(params)
	instance.SetupDmrClusterMocks(params, "")

	clusterUrl := instance.baseUrl + "/api/v2/missionControl/eventBrokerServices/" + params.ServiceId + "/broker/cluster"

	resp, err := http.Post(clusterUrl, "application/json", strings.NewReader(`{"clusterName":"named-by-test"}`))
	if err != nil {
		t.Fatalf("enable request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 202 {
		t.Fatalf("expected 202 from the enable request, got %d", resp.StatusCode)
	}

	// The enable request re-registers the service responders, so a read now reports a cluster.
	serviceResp, err := http.Get(instance.baseUrl + "/api/v2/missionControl/eventBrokerServices/" + params.ServiceId)
	if err != nil {
		t.Fatalf("service read failed: %v", err)
	}
	defer func() { _ = serviceResp.Body.Close() }()
	body, err := io.ReadAll(serviceResp.Body)
	if err != nil {
		t.Fatalf("could not read the service response: %v", err)
	}

	data := decodeServiceResponse(t, string(body))
	broker := data["broker"].(map[string]interface{})
	cluster, ok := broker["cluster"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected the service to report a cluster after enabling, got %v", broker["cluster"])
	}
	if cluster["name"] != "named-by-test" {
		t.Errorf("expected the cluster name from the request, got %v", cluster["name"])
	}
}

// TestSetupDmrClusterMocksReportsAFailedOperation verifies that the helper can simulate a
// refusal, which the provider must surface rather than treat as success.
func TestSetupDmrClusterMocksReportsAFailedOperation(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	instance := &TestInstance{baseUrl: "https://api.solace.test", mockedApi: true}
	params := ConfigurableParams{ServiceId: "svc-mock-dmr-fail", ServiceName: "mock-dmr-fail"}
	instance.SetupDmrClusterMocks(params, "refused by the event mesh")

	operationUrl := instance.baseUrl + "/api/v2/missionControl/eventBrokerServices/" +
		params.ServiceId + "/operations/dmr-operation-123"

	resp, err := http.Get(operationUrl)
	if err != nil {
		t.Fatalf("operation read failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read the operation response: %v", err)
	}

	data := decodeServiceResponse(t, string(body))
	if data["status"] != "FAILED" {
		t.Errorf("expected a FAILED operation, got %v", data["status"])
	}
	operationError, ok := data["error"].(map[string]interface{})
	if !ok || operationError["message"] != "refused by the event mesh" {
		t.Errorf("expected the operation to carry the refusal message, got %v", data["error"])
	}
}

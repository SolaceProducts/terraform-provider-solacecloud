package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"

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

func readMockedMateLinkState(t *testing.T, instance *TestInstance, serviceId string) bool {
	t.Helper()
	resp, err := http.Get(instance.baseUrl + "/api/v2/missionControl/eventBrokerServices/" + serviceId)
	if err != nil {
		t.Fatalf("service read failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read the service response: %v", err)
	}
	broker := decodeServiceResponse(t, string(body))["broker"].(map[string]interface{})
	enabled, ok := broker["redundancyGroupSslEnabled"].(bool)
	if !ok {
		t.Fatalf("expected the service to report redundancyGroupSslEnabled, got %v", broker["redundancyGroupSslEnabled"])
	}
	return enabled
}

func patchMateLink(t *testing.T, url, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("PATCH", url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("could not build the mate-link request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("mate-link request failed: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func TestSetupMateLinkMocksAppliesTheRequestedState(t *testing.T) {
	cases := []struct {
		name        string
		mateLinkOff bool
		body        string
		want        bool
	}{
		{name: "enable", mateLinkOff: true, body: `{"enabled":true}`, want: true},
		{name: "disable", mateLinkOff: false, body: `{"enabled":false}`, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			instance := &TestInstance{baseUrl: "https://api.solace.test", mockedApi: true}
			params := ConfigurableParams{ServiceId: "svc-mock-mlk", ServiceName: "mock-mlk", MateLinkOff: tc.mateLinkOff}
			instance.SetupDefaultMocks(params)
			instance.SetupMateLinkMocks(params, "")

			if got := readMockedMateLinkState(t, instance, params.ServiceId); got != !tc.mateLinkOff {
				t.Fatalf("expected the service to start with mate-link encryption %t, got %t", !tc.mateLinkOff, got)
			}

			resp := patchMateLink(t, instance.baseUrl+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/mateLinkEncryption", tc.body)
			if resp.StatusCode != 202 {
				t.Fatalf("expected 202 from the mate-link request, got %d", resp.StatusCode)
			}
			if got := readMockedMateLinkState(t, instance, params.ServiceId); got != tc.want {
				t.Errorf("expected mate-link encryption %t after the request, got %t", tc.want, got)
			}
		})
	}
}

func TestSetupMateLinkMocksFailedOperationKeepsTheState(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	instance := &TestInstance{baseUrl: "https://api.solace.test", mockedApi: true}
	params := ConfigurableParams{ServiceId: "svc-mock-mlk-fail", ServiceName: "mock-mlk-fail", MateLinkOff: true}
	instance.SetupDefaultMocks(params)
	instance.SetupMateLinkMocks(params, "service is not highly available")

	resp := patchMateLink(t, instance.baseUrl+"/api/v2/missionControl/eventBrokerServices/"+params.ServiceId+"/mateLinkEncryption", `{"enabled":true}`)
	if resp.StatusCode != 202 {
		t.Fatalf("expected 202 from the mate-link request, got %d", resp.StatusCode)
	}
	if readMockedMateLinkState(t, instance, params.ServiceId) {
		t.Error("expected mate-link encryption to stay disabled after a failed operation")
	}

	opResp, err := http.Get(instance.baseUrl + "/api/v2/missionControl/eventBrokerServices/" + params.ServiceId + "/operations/mate-link-operation-123")
	if err != nil {
		t.Fatalf("operation read failed: %v", err)
	}
	defer func() { _ = opResp.Body.Close() }()
	body, err := io.ReadAll(opResp.Body)
	if err != nil {
		t.Fatalf("could not read the operation response: %v", err)
	}
	data := decodeServiceResponse(t, string(body))
	if data["status"] != "FAILED" {
		t.Errorf("expected a FAILED operation, got %v", data["status"])
	}
	operationError, ok := data["error"].(map[string]interface{})
	if !ok || operationError["message"] != "service is not highly available" {
		t.Errorf("expected the operation to carry the refusal message, got %v", data["error"])
	}
}

func TestSetupMateLinkMocksRejectsARequestWithoutEnabled(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	instance := &TestInstance{baseUrl: "https://api.solace.test", mockedApi: true}
	instance.SetupMateLinkMocks(ConfigurableParams{}, "")

	resp := patchMateLink(t, instance.baseUrl+"/api/v2/missionControl/eventBrokerServices/6q1p55o6ovr/mateLinkEncryption", `{}`)
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for a request without enabled, got %d", resp.StatusCode)
	}
}

func TestReadMateLinkEnabled(t *testing.T) {
	cases := []struct {
		name        string
		body        io.Reader
		wantEnabled bool
		wantOk      bool
	}{
		{name: "no body", body: nil},
		{name: "unreadable body", body: iotest.ErrReader(io.ErrUnexpectedEOF)},
		{name: "not JSON", body: strings.NewReader(`enabled=true`)},
		{name: "enabled missing", body: strings.NewReader(`{}`)},
		{name: "enabled null", body: strings.NewReader(`{"enabled":null}`)},
		{name: "enabled true", body: strings.NewReader(`{"enabled":true}`), wantEnabled: true, wantOk: true},
		{name: "enabled false", body: strings.NewReader(`{"enabled":false}`), wantEnabled: false, wantOk: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("PATCH", "https://api.solace.test", tc.body)
			if err != nil {
				t.Fatalf("could not build the request: %v", err)
			}
			enabled, ok := readMateLinkEnabled(req)
			if enabled != tc.wantEnabled || ok != tc.wantOk {
				t.Errorf("expected (%t, %t), got (%t, %t)", tc.wantEnabled, tc.wantOk, enabled, ok)
			}
		})
	}
}

func TestSetupLdapManagementAccessMocksFollowsTheApiLifecycle(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	instance := &TestInstance{baseUrl: "https://api.solace.test", mockedApi: true}
	mock := instance.SetupLdapManagementAccessMocks("svc-mock-ldap")
	url := instance.baseUrl + "/api/v2/missionControl/eventBrokerServices/svc-mock-ldap/ldapManagementAccess"
	send := func(method, body string) int {
		req, err := http.NewRequest(method, url, strings.NewReader(body))
		if err != nil {
			t.Fatalf("could not build the %s request: %v", method, err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s request failed: %v", method, err)
		}
		_ = resp.Body.Close()
		return resp.StatusCode
	}

	steps := []struct {
		method, body string
		want         int
	}{
		{"GET", "", 404},
		{"PATCH", `{"enabled":false}`, 404},
		{"DELETE", "", 404},
		{"POST", "not json", 400},
		{"POST", `{"enabled":true,"groupAttributeName":"memberOf"}`, 202},
		{"POST", `{"enabled":true,"groupAttributeName":"memberOf"}`, 409},
		{"PATCH", `{}`, 400},
		{"PATCH", `{"enabled":false}`, 202},
		{"GET", "", 200},
		{"DELETE", "", 202},
		{"GET", "", 404},
	}
	for i, s := range steps {
		if got := send(s.method, s.body); got != s.want {
			t.Fatalf("step %d: %s %s returned %d, want %d", i, s.method, s.body, got, s.want)
		}
	}

	if patches := mock.Patches(); len(patches) != 1 || string(patches[0]["enabled"]) != "false" {
		t.Errorf("expected one recorded PATCH that sets enabled to false, got %v", patches)
	}

	send("POST", `{"enabled":true}`)
	if !mock.Configured() {
		t.Fatal("expected the configuration to exist after a POST")
	}
	mock.Remove()
	if mock.Configured() || send("GET", "") != 404 {
		t.Error("expected Remove to delete the configuration")
	}
}

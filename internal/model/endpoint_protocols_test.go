package model

import (
	"testing"
	"terraform-provider-solacecloud/missioncontrol"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestEndpointProtocolSchema(t *testing.T) {
	schema := EndpointProtocolSchema()

	// Verify schema is not nil
	assert.NotNil(t, schema, "Schema should be a valid attribute")

	// Verify schema is computed
	assert.True(t, schema.IsComputed())
	assert.False(t, schema.IsRequired())
	assert.False(t, schema.IsOptional())

	// Verify markdown description is present
	assert.NotEmpty(t, schema.GetMarkdownDescription())
}

func TestConnectionEndpointProtocolSchema(t *testing.T) {
	schema := ConnectionEndpointProtocolSchema()

	// Verify it's a SingleNestedAttribute
	assert.NotNil(t, schema)

	// Verify schema is required
	assert.True(t, schema.IsRequired())
	assert.False(t, schema.IsComputed())
	assert.False(t, schema.IsOptional())

	// Verify markdown description is present
	assert.NotEmpty(t, schema.GetMarkdownDescription())
}

func TestEndpointProtocolsTypes(t *testing.T) {
	types := EndpointProtocolsTypes()

	// Verify all expected protocol keys are present
	expectedProtocols := []string{
		"web",
		"management_tls",
		"rest_incoming_tls",
		"amqp",
		"mqtt_websocket",
		"rest_incoming",
		"web_tls",
		"smf_compressed",
		"mqtt",
		"smf",
		"amqp_tls",
		"mqtt_tls",
		"smf_tls",
		"mqtt_websocket_tls",
		"ssh_tls",
	}

	for _, protocol := range expectedProtocols {
		assert.Contains(t, types, protocol, "Missing protocol: %s", protocol)
		assert.NotNil(t, types[protocol], "Protocol type should not be nil: %s", protocol)
	}

	// Verify correct number of types
	assert.Len(t, types, 15)
}

func TestConnectionEndpointProtocolsTypes(t *testing.T) {
	types := ConnectionEndpointProtocolsTypes()

	// Verify all expected protocol keys are present
	expectedProtocols := []string{
		"web",
		"management_tls",
		"rest_incoming_tls",
		"amqp",
		"mqtt_websocket",
		"rest_incoming",
		"web_tls",
		"smf_compressed",
		"mqtt",
		"smf",
		"amqp_tls",
		"mqtt_tls",
		"smf_tls",
		"mqtt_websocket_tls",
		"ssh_tls",
	}

	for _, protocol := range expectedProtocols {
		assert.Contains(t, types, protocol, "Missing protocol: %s", protocol)
		assert.NotNil(t, types[protocol], "Protocol type should not be nil: %s", protocol)
	}

	// Verify correct number of types
	assert.Len(t, types, 15)
}

func TestToObjectValue_WithAllProtocols(t *testing.T) {
	// Create a full set of ports
	port8080 := int32(8080)
	port8443 := int32(8443)
	port5672 := int32(5672)
	port5671 := int32(5671)
	port1883 := int32(1883)
	port8883 := int32(8883)
	port8000 := int32(8000)
	port8001 := int32(8001)
	port9000 := int32(9000)
	port9001 := int32(9001)
	port9002 := int32(9002)
	port9003 := int32(9003)
	port9004 := int32(9004)
	port9943 := int32(9943)
	port2222 := int32(2222)

	ports := []missioncontrol.ServiceConnectionEndpointPort{
		{Protocol: "serviceWebPlainTextListenPort", Port: &port8080},
		{Protocol: "serviceWebTlsListenPort", Port: &port8443},
		{Protocol: "serviceAmqpPlainTextListenPort", Port: &port5672},
		{Protocol: "serviceAmqpTlsListenPort", Port: &port5671},
		{Protocol: "serviceMqttPlainTextListenPort", Port: &port1883},
		{Protocol: "serviceMqttTlsListenPort", Port: &port8883},
		{Protocol: "serviceMqttWebSocketListenPort", Port: &port8000},
		{Protocol: "serviceMqttTlsWebSocketListenPort", Port: &port8001},
		{Protocol: "serviceRestIncomingPlainTextListenPort", Port: &port9000},
		{Protocol: "serviceRestIncomingTlsListenPort", Port: &port9001},
		{Protocol: "serviceSmfPlainTextListenPort", Port: &port9002},
		{Protocol: "serviceSmfCompressedListenPort", Port: &port9003},
		{Protocol: "serviceSmfTlsListenPort", Port: &port9004},
		{Protocol: "serviceManagementTlsListenPort", Port: &port9943},
		{Protocol: "managementSshTlsListenPort", Port: &port2222},
	}

	objValue, diags := ToObjectValue(ports)

	// Should not have any diagnostics
	assert.False(t, diags.HasError())
	assert.Empty(t, diags)

	// Should not be null or unknown
	assert.False(t, objValue.IsNull())
	assert.False(t, objValue.IsUnknown())

	// Verify all protocols are set
	attrs := objValue.Attributes()

	// Verify web protocol
	webAttr := attrs["web"]
	assert.False(t, webAttr.IsNull())
	webObj := webAttr.(types.Object)
	assert.Equal(t, types.Int64Value(8080), webObj.Attributes()["port"])

	// Verify web_tls protocol
	webTlsAttr := attrs["web_tls"]
	assert.False(t, webTlsAttr.IsNull())
	webTlsObj := webTlsAttr.(types.Object)
	assert.Equal(t, types.Int64Value(8443), webTlsObj.Attributes()["port"])

	// Verify amqp protocol
	amqpAttr := attrs["amqp"]
	assert.False(t, amqpAttr.IsNull())
	amqpObj := amqpAttr.(types.Object)
	assert.Equal(t, types.Int64Value(5672), amqpObj.Attributes()["port"])

	// Verify management_tls protocol
	mgmtAttr := attrs["management_tls"]
	assert.False(t, mgmtAttr.IsNull())
	mgmtObj := mgmtAttr.(types.Object)
	assert.Equal(t, types.Int64Value(9943), mgmtObj.Attributes()["port"])

	// Verify ssh_tls protocol
	sshAttr := attrs["ssh_tls"]
	assert.False(t, sshAttr.IsNull())
	sshObj := sshAttr.(types.Object)
	assert.Equal(t, types.Int64Value(2222), sshObj.Attributes()["port"])
}

func TestToObjectValue_WithDisabledProtocols(t *testing.T) {
	// Create ports with some disabled (port 0)
	port8080 := int32(8080)
	port0 := int32(0)

	ports := []missioncontrol.ServiceConnectionEndpointPort{
		{Protocol: "serviceWebPlainTextListenPort", Port: &port8080},
		{Protocol: "serviceWebTlsListenPort", Port: &port0}, // Disabled
		{Protocol: "serviceAmqpPlainTextListenPort", Port: &port0}, // Disabled
	}

	objValue, diags := ToObjectValue(ports)

	// Should not have any diagnostics
	assert.False(t, diags.HasError())
	assert.Empty(t, diags)

	// Should not be null
	assert.False(t, objValue.IsNull())

	attrs := objValue.Attributes()

	// Enabled protocol should have a value
	webAttr := attrs["web"]
	assert.False(t, webAttr.IsNull())
	webObj := webAttr.(types.Object)
	assert.Equal(t, types.Int64Value(8080), webObj.Attributes()["port"])

	// Disabled protocols should be null
	assert.True(t, attrs["web_tls"].IsNull())
	assert.True(t, attrs["amqp"].IsNull())
}

func TestToObjectValue_WithEmptyPorts(t *testing.T) {
	// Create empty ports list
	ports := []missioncontrol.ServiceConnectionEndpointPort{}

	objValue, diags := ToObjectValue(ports)

	// Should not have any diagnostics
	assert.False(t, diags.HasError())
	assert.Empty(t, diags)

	// Should not be null
	assert.False(t, objValue.IsNull())

	// All protocols should be null
	attrs := objValue.Attributes()
	for protocolName, attr := range attrs {
		assert.True(t, attr.IsNull(), "Protocol %s should be null", protocolName)
	}
}

func TestToObjectValue_WithUnknownProtocol(t *testing.T) {
	// Create ports with an unknown protocol (should be ignored)
	port8080 := int32(8080)
	port9999 := int32(9999)

	ports := []missioncontrol.ServiceConnectionEndpointPort{
		{Protocol: "serviceWebPlainTextListenPort", Port: &port8080},
		{Protocol: "unknownProtocol", Port: &port9999}, // Should be ignored
	}

	objValue, diags := ToObjectValue(ports)

	// Should not have any diagnostics
	assert.False(t, diags.HasError())
	assert.Empty(t, diags)

	// Should not be null
	assert.False(t, objValue.IsNull())

	attrs := objValue.Attributes()

	// Known protocol should be set
	webAttr := attrs["web"]
	assert.False(t, webAttr.IsNull())
	webObj := webAttr.(types.Object)
	assert.Equal(t, types.Int64Value(8080), webObj.Attributes()["port"])

	// Unknown protocol should not affect other protocols (they should remain null)
	assert.True(t, attrs["web_tls"].IsNull())
}

func TestGetAPIProtocolName(t *testing.T) {
	testCases := []struct {
		terraformName string
		expectedAPI   string
	}{
		{"web", "serviceWebPlainTextListenPort"},
		{"web_tls", "serviceWebTlsListenPort"},
		{"management_tls", "serviceManagementTlsListenPort"},
		{"rest_incoming_tls", "serviceRestIncomingTlsListenPort"},
		{"amqp", "serviceAmqpPlainTextListenPort"},
		{"amqp_tls", "serviceAmqpTlsListenPort"},
		{"mqtt_websocket", "serviceMqttWebSocketListenPort"},
		{"rest_incoming", "serviceRestIncomingPlainTextListenPort"},
		{"smf_compressed", "serviceSmfCompressedListenPort"},
		{"mqtt", "serviceMqttPlainTextListenPort"},
		{"smf", "serviceSmfPlainTextListenPort"},
		{"mqtt_tls", "serviceMqttTlsListenPort"},
		{"smf_tls", "serviceSmfTlsListenPort"},
		{"mqtt_websocket_tls", "serviceMqttTlsWebSocketListenPort"},
		{"ssh_tls", "managementSshTlsListenPort"},
	}

	for _, tc := range testCases {
		t.Run(tc.terraformName, func(t *testing.T) {
			result := GetAPIProtocolName(tc.terraformName)
			assert.Equal(t, tc.expectedAPI, result)
		})
	}
}

func TestGetAPIProtocolName_UnknownProtocol(t *testing.T) {
	// Unknown protocol should return the original name as fallback
	result := GetAPIProtocolName("unknown_protocol")
	assert.Equal(t, "unknown_protocol", result)
}

func TestGetAPIProtocolName_RoundTrip(t *testing.T) {
	// Test that all protocols in the mapping can be converted back and forth
	terraformNames := []string{
		"web", "management_tls", "rest_incoming_tls", "amqp",
		"mqtt_websocket", "rest_incoming", "web_tls", "smf_compressed",
		"mqtt", "smf", "amqp_tls", "mqtt_tls", "smf_tls",
		"mqtt_websocket_tls", "ssh_tls",
	}

	// Create a map from API names back to Terraform names for validation
	apiToTerraform := map[string]string{
		"serviceWebPlainTextListenPort":          "web",
		"serviceManagementTlsListenPort":         "management_tls",
		"serviceRestIncomingTlsListenPort":       "rest_incoming_tls",
		"serviceAmqpPlainTextListenPort":         "amqp",
		"serviceMqttWebSocketListenPort":         "mqtt_websocket",
		"serviceRestIncomingPlainTextListenPort": "rest_incoming",
		"serviceWebTlsListenPort":                "web_tls",
		"serviceSmfCompressedListenPort":         "smf_compressed",
		"serviceMqttPlainTextListenPort":         "mqtt",
		"serviceSmfPlainTextListenPort":          "smf",
		"serviceAmqpTlsListenPort":               "amqp_tls",
		"serviceMqttTlsListenPort":               "mqtt_tls",
		"serviceSmfTlsListenPort":                "smf_tls",
		"serviceMqttTlsWebSocketListenPort":      "mqtt_websocket_tls",
		"managementSshTlsListenPort":             "ssh_tls",
	}

	for _, tfName := range terraformNames {
		apiName := GetAPIProtocolName(tfName)
		assert.NotEmpty(t, apiName, "API name should not be empty for %s", tfName)

		// Verify the round trip
		assert.Equal(t, tfName, apiToTerraform[apiName],
			"Round trip failed for %s -> %s", tfName, apiName)
	}
}
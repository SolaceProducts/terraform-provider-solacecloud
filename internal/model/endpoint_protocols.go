package model

import (
	"terraform-provider-solacecloud/missioncontrol"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Package model contains the EndpointProtocols schema which is an Object nested into the ConnectionEndpoint object.
// It represents the set of protocols that are served by the connection endpoint.  Disabled protocols have their
// attribute field set to null, otherwise the EndpointProtocol object will contain the port over which the protocol
// service is listening on.

func EndpointProtocolSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "<p>The protocols and port numbers of the connection endpoint. " +
			"All messaging and management protocols along with the port numbers must be specified in the request." +
			"</p>\n" +
			"<p>Connection specific protocols. </p>\n" +
			"<ul>\n" +
			"  <li><b>Solace Messaging</b>\n" +
			"    <ul>\n" +
			"      <li>'smf' - Use SMF Host (plain-text) over TCP to connect and exchange " +
			"           messages with the event broker service.</li>\n" +
			"      <li>'smf_compressed' - Use SMF (plain-text) in a compressed format over TCP to " +
			"           connect and exchange messages with the event broker service.</li>\n" +
			"      <li>'smf_tls' - Use secure SMF using TLS over TCP.</li>\n" +
			"    </ul>\n" +
			"  </li>\n" +
			"  <br>\n" +
			"  <li><b>Solace Web Messaging</b>\n" +
			"    <ul>\n" +
			"      <li>'web' - Use WebSocket over HTTP (plain-text).</li>\n" +
			"      <li>'web_tls' - Use WebSocket over secured HTTP.</li>\n" +
			"    </ul>\n" +
			"  </li>\n" +
			"  <br>\n" +
			"  <li><b>AMQP</b>\n" +
			"    <ul>\n" +
			"      <li>'amqp' - Use AMQP (plain-text).</li>\n" +
			"      <li>'amqp_tls' - Use AMQP over a secure TCP connection.</li>\n" +
			"    </ul>\n" +
			"  </li>\n" +
			"  <br>\n" +
			"  <li><b>MQTT</b>\n" +
			"    <ul>\n" +
			"      <li>'mqtt' - Use MQTT (plain-text).</li>\n" +
			"      <li>'mqtt_websocket' - Use MQTT WebSocket (plain-text).</li>\n" +
			"      <li>'mqtt_tls' - Use secure MQTT.</li>\n" +
			"      <li>'mqtt_websocket_tls' - Use WebSocket secured MQTT.</li>\n" +
			"    </ul>\n" +
			"  </li>\n" +
			"  <br>\n" +
			"  <li><b>REST</b>\n" +
			"    <ul>\n" +
			"      <li>'rest_incoming' - Use REST messaging (plain-text).</li>\n" +
			"      <li>'rest_incoming_tls' - Use secure REST messaging.</li>\n" +
			"    </ul>\n" +
			"  </li>\n" +
			"  <br>\n" +
			"  <li><b>Management</b>\n" +
			"    <ul>\n" +
			"      <li>'management_tls' - Use the secured management connection, which uses SEMP to " +
			"           manage the event broker. This port must be enabled on at least one of the service connection " +
			"           endpoints on the event broker service.</li>\n" +
			"      <li>'ssh_tls' - Use a secure port to connect to the event broker service to issue " +
			"           Solace Command Line Interface (CLI). This port provides you with scope-restricted access to the " +
			"           event broker service.</li>\n" +
			"    </ul>\n" +
			"  </li>\n" +
			"</ul>",
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"web":                EndpointProtocolModelType(),
			"management_tls":     EndpointProtocolModelType(),
			"rest_incoming_tls":  EndpointProtocolModelType(),
			"amqp":               EndpointProtocolModelType(),
			"mqtt_websocket":     EndpointProtocolModelType(),
			"rest_incoming":      EndpointProtocolModelType(),
			"web_tls":            EndpointProtocolModelType(),
			"smf_compressed":     EndpointProtocolModelType(),
			"mqtt":               EndpointProtocolModelType(),
			"smf":                EndpointProtocolModelType(),
			"amqp_tls":           EndpointProtocolModelType(),
			"mqtt_tls":           EndpointProtocolModelType(),
			"smf_tls":            EndpointProtocolModelType(),
			"mqtt_websocket_tls": EndpointProtocolModelType(),
			"ssh_tls":            EndpointProtocolModelType(),
		},
	}
}

func EndpointProtocolsTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"web":                EndpointProtocolModelType().GetType(),
		"management_tls":     EndpointProtocolModelType().GetType(),
		"rest_incoming_tls":  EndpointProtocolModelType().GetType(),
		"amqp":               EndpointProtocolModelType().GetType(),
		"mqtt_websocket":     EndpointProtocolModelType().GetType(),
		"rest_incoming":      EndpointProtocolModelType().GetType(),
		"web_tls":            EndpointProtocolModelType().GetType(),
		"smf_compressed":     EndpointProtocolModelType().GetType(),
		"mqtt":               EndpointProtocolModelType().GetType(),
		"smf":                EndpointProtocolModelType().GetType(),
		"amqp_tls":           EndpointProtocolModelType().GetType(),
		"mqtt_tls":           EndpointProtocolModelType().GetType(),
		"smf_tls":            EndpointProtocolModelType().GetType(),
		"mqtt_websocket_tls": EndpointProtocolModelType().GetType(),
		"ssh_tls":            EndpointProtocolModelType().GetType(),
	}
}

func ToObjectValue(Ports []missioncontrol.ServiceConnectionEndpointPort) (basetypes.ObjectValue, diag.Diagnostics) {
	// Maps our APIs protocol names to Terraform's friendly attribute names (lowercase and underscores only - Plus make the name succinct and nice by avoiding redundant parts such as service, listen port, or plain text)
	attributeMapping := map[string]string{
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

	values := map[string]attr.Value{
		"web":                NullEndpointProtocol(),
		"management_tls":     NullEndpointProtocol(),
		"rest_incoming_tls":  NullEndpointProtocol(),
		"amqp":               NullEndpointProtocol(),
		"mqtt_websocket":     NullEndpointProtocol(),
		"rest_incoming":      NullEndpointProtocol(),
		"web_tls":            NullEndpointProtocol(),
		"smf_compressed":     NullEndpointProtocol(),
		"mqtt":               NullEndpointProtocol(),
		"smf":                NullEndpointProtocol(),
		"amqp_tls":           NullEndpointProtocol(),
		"mqtt_tls":           NullEndpointProtocol(),
		"smf_tls":            NullEndpointProtocol(),
		"mqtt_websocket_tls": NullEndpointProtocol(),
		"ssh_tls":            NullEndpointProtocol(),
	}

	for _, port := range Ports {
		// Leave disabled as null as this is a better way to represent the absence of something in terraform.
		if *port.Port == 0 {
			continue
		}

		// Map each protocol from the API Response's list to its corresponding object's attribute
		// More work than if we had chosen a Nested Map Attribute, but this is worth it for the end user as the schema
		// will describe which protocols exists.
		protocolName := string(port.Protocol)
		if attributeName, ok := attributeMapping[protocolName]; ok {
			portModel, diags := EndpointProtocolModel{
				Port: types.Int64Value(int64(*port.Port)),
			}.ToObjectValue()
			if diags.HasError() {
				return basetypes.NewObjectUnknown(EndpointProtocolsTypes()), diags
			}
			values[attributeName] = portModel
		}
	}

	return types.ObjectValue(
		EndpointProtocolsTypes(),
		values,
	)
}

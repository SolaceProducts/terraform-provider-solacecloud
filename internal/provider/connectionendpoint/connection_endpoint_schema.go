package connectionendpoint

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ConnectionEndpointSchema defines the schema for the Solace Cloud Connection Endpoint resource
func ConnectionEndpointSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"service_id": schema.StringAttribute{
			Required:    true,
			Description: "The ID of the Solace Cloud Service this connection endpoint belongs to.",
		},
		"name": schema.StringAttribute{
			Required:    true,
			Description: "The name of the connection endpoint.",
		},
		"description": schema.StringAttribute{
			Optional:    true,
			Description: "A description for the connection endpoint.",
		},
		"access_type": schema.StringAttribute{
			Required:    true,
			Description: "The access type for the connection endpoint (e.g., PUBLIC or PRIVATE).",
		},
		"ports": schema.MapAttribute{
			Required:    true,
			ElementType: types.Int64Type,
			Description: "A map of port names to their respective values.",
		},
	}
}

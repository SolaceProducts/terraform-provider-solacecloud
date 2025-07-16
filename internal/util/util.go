package util

import "github.com/hashicorp/terraform-plugin-framework/attr"

func IsKnown(value attr.Value) bool {
	return !value.IsNull() && !value.IsUnknown()
}

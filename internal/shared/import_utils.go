// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package shared

import (
	"strings"
)

// ParseImportId parses an import ID by splitting on "/" and validating the expected number of parts
func ParseImportId(id string, expectedParts int) []string {
	parts := strings.Split(id, "/")
	if len(parts) != expectedParts {
		return nil
	}
	return parts
}
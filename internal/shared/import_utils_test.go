package shared

import (
	"reflect"
	"testing"
)

func TestParseImportId(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		expectedParts int
		expectedResult []string
	}{
		{
			name:          "valid single part",
			id:            "resource-id",
			expectedParts: 1,
			expectedResult: []string{"resource-id"},
		},
		{
			name:          "valid two parts",
			id:            "parent-id/child-id",
			expectedParts: 2,
			expectedResult: []string{"parent-id", "child-id"},
		},
		{
			name:          "valid three parts",
			id:            "org-id/service-id/resource-id",
			expectedParts: 3,
			expectedResult: []string{"org-id", "service-id", "resource-id"},
		},
		{
			name:          "valid multiple parts with special characters",
			id:            "org-123/service_456/resource.789",
			expectedParts: 3,
			expectedResult: []string{"org-123", "service_456", "resource.789"},
		},
		{
			name:          "mismatch - too few parts",
			id:            "single-part",
			expectedParts: 2,
			expectedResult: nil,
		},
		{
			name:          "mismatch - too many parts",
			id:            "part1/part2/part3",
			expectedParts: 2,
			expectedResult: nil,
		},
		{
			name:          "empty string with expected 1 part",
			id:            "",
			expectedParts: 1,
			expectedResult: []string{""},
		},
		{
			name:          "empty string with expected 2 parts",
			id:            "",
			expectedParts: 2,
			expectedResult: nil,
		},
		{
			name:          "single slash creates two empty parts",
			id:            "/",
			expectedParts: 2,
			expectedResult: []string{"", ""},
		},
		{
			name:          "trailing slash",
			id:            "part1/part2/",
			expectedParts: 3,
			expectedResult: []string{"part1", "part2", ""},
		},
		{
			name:          "leading slash",
			id:            "/part1/part2",
			expectedParts: 3,
			expectedResult: []string{"", "part1", "part2"},
		},
		{
			name:          "multiple consecutive slashes",
			id:            "part1//part2",
			expectedParts: 3,
			expectedResult: []string{"part1", "", "part2"},
		},
		{
			name:          "complex real-world example",
			id:            "abc123/def456/ghi789/jkl012",
			expectedParts: 4,
			expectedResult: []string{"abc123", "def456", "ghi789", "jkl012"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseImportId(tt.id, tt.expectedParts)

			if !reflect.DeepEqual(result, tt.expectedResult) {
				t.Errorf("ParseImportId(%q, %d) = %v, want %v",
					tt.id, tt.expectedParts, result, tt.expectedResult)
			}
		})
	}
}
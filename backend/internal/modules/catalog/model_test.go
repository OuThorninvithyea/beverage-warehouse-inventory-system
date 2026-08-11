package catalog

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCategoryAndProductResponsesKeepNullableFields(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "category parent", value: Category{}, want: `"parent_id":null`},
		{name: "product category", value: Product{}, want: `"category_id":null`},
		{name: "product barcode", value: Product{}, want: `"barcode":null`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, err := json.Marshal(test.value)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if !strings.Contains(string(body), test.want) {
				t.Fatalf("JSON = %s, want %s", body, test.want)
			}
		})
	}
}

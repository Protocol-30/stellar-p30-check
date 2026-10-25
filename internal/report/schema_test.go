package report

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Protocol-30/stellar-p30-check/internal/deps"
)

// schemaContract is intentionally not a JSON Schema evaluator. This check guards
// the published document's field and enum contract against accidental drift from
// the Go model; JSON output behavior is exercised separately in report_test.go.
type schemaContract struct {
	Schema               string                    `json:"$schema"`
	Type                 string                    `json:"type"`
	AdditionalProperties bool                      `json:"additionalProperties"`
	Properties           map[string]schemaContract `json:"properties"`
	Required             []string                  `json:"required"`
	Defs                 map[string]schemaContract `json:"$defs"`
	Enum                 []string                  `json:"enum"`
	Const                string                    `json:"const"`
	Minimum              int                       `json:"minimum"`
}

func TestPublishedSchemaMatchesModelContract(t *testing.T) {
	b, err := os.ReadFile("../../docs/report.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema schemaContract
	if err := json.Unmarshal(b, &schema); err != nil {
		t.Fatal(err)
	}
	if schema.Schema != "https://json-schema.org/draft/2020-12/schema" {
		t.Errorf("unexpected schema dialect %q", schema.Schema)
	}
	models := []struct {
		name   string
		model  any
		schema schemaContract
	}{
		{"report", ScanReport{}, schema},
		{"finding", Finding{}, schema.Defs["finding"]},
		{"summary", Summary{}, schema.Defs["summary"]},
		{"inventory_item", InventoryItem{}, schema.Defs["inventory_item"]},
		{"skipped", Skipped{}, schema.Defs["skipped"]},
		{"coverage", Coverage{}, schema.Defs["coverage"]},
		{"dependency", deps.Dependency{}, schema.Defs["dependency"]},
		{"capabilities", deps.Capabilities{}, schema.Defs["capabilities"]},
		{"dependency_warning", deps.Warning{}, schema.Defs["dependency_warning"]},
	}
	for _, model := range models {
		t.Run(model.name, func(t *testing.T) {
			if model.schema.Type != "object" || model.schema.AdditionalProperties {
				t.Error("model must be a closed object")
			}
			typ := reflect.TypeOf(model.model)
			if len(model.schema.Properties) != typ.NumField() {
				t.Errorf("schema has %d properties; Go type has %d fields", len(model.schema.Properties), typ.NumField())
			}
			var required []string
			for i := range typ.NumField() {
				field := typ.Field(i)
				tag := strings.Split(field.Tag.Get("json"), ",")
				name := tag[0]
				if _, ok := model.schema.Properties[name]; !ok {
					t.Errorf("Go field %s missing from schema", name)
				}
				if !slices.Contains(tag[1:], "omitempty") {
					required = append(required, name)
				}
			}
			slices.Sort(required)
			actual := slices.Clone(model.schema.Required)
			slices.Sort(actual)
			if !slices.Equal(actual, required) {
				t.Errorf("required fields = %v; want %v", actual, required)
			}
		})
	}
	if schema.Properties["schema_version"].Const != SchemaVersion {
		t.Errorf("schema version does not match %q", SchemaVersion)
	}
	finding := schema.Defs["finding"]
	if !slices.Equal(finding.Properties["severity"].Enum, []string{string(SeverityInfo), string(SeverityMedium), string(SeverityHigh)}) {
		t.Error("published severity enum differs from model")
	}
	if !slices.Equal(finding.Properties["confidence"].Enum, []string{string(ConfidenceLow), string(ConfidenceMedium), string(ConfidenceHigh)}) {
		t.Error("published confidence enum differs from model")
	}
	if finding.Properties["line"].Minimum != 1 {
		t.Error("a present line number must be positive")
	}
}

func TestCapabilitySchemaEnumsAndLegacySchema(t *testing.T) {
	data, err := os.ReadFile("../../docs/report.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema schemaContract
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cap88", "muxed_contract_xdr", "w_address_helpers"} {
		if !slices.Equal(schema.Defs["capabilities"].Properties[name].Enum, []string{"supported", "unsupported", "unknown"}) {
			t.Errorf("wrong capability state contract for %s", name)
		}
	}
	legacy, err := os.ReadFile("../../docs/report-v1.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var old schemaContract
	if err := json.Unmarshal(legacy, &old); err != nil {
		t.Fatal(err)
	}
	if old.Properties["schema_version"].Const != "1" {
		t.Error("legacy report schema was not preserved")
	}
	if _, exists := old.Properties["dependencies"]; exists {
		t.Error("legacy schema unexpectedly contains version2 fields")
	}
}

package management

import (
	"encoding/json"
	"testing"
)

func TestPrometheusQueryDataValue(t *testing.T) {
	var vector prometheusQueryData
	if err := json.Unmarshal([]byte(`{"resultType":"vector","result":[{"value":[1710000000,"12.5"]}]}`), &vector); err != nil {
		t.Fatal(err)
	}
	if vector.value() != "12.5" {
		t.Fatalf("unexpected vector value: %q", vector.value())
	}
	var scalar prometheusQueryData
	if err := json.Unmarshal([]byte(`{"resultType":"scalar","result":[1710000000,"3"]}`), &scalar); err != nil {
		t.Fatal(err)
	}
	if scalar.value() != "3" {
		t.Fatalf("unexpected scalar value: %q", scalar.value())
	}
}

package rewrite

import (
	"encoding/json"
	"testing"
)

func TestInjectAutoMode_NewFeature(t *testing.T) {
	body := []byte(`{"features":{"some_other_flag":{"defaultValue":true}}}`)
	out, changed, err := InjectAutoMode(body)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	var resp map[string]any
	json.Unmarshal(out, &resp)
	features := resp["features"].(map[string]any)
	ac := features["tengu_auto_mode_config"].(map[string]any)
	dv := ac["defaultValue"].(map[string]any)
	if dv["enabled"] != "enabled" {
		t.Fatalf("got enabled %v", dv["enabled"])
	}
	models := dv["allowModels"].([]any)
	if len(models) != 1 || models[0] != "*" {
		t.Fatalf("got allowModels %v", models)
	}
}

func TestInjectAutoMode_ExistingFeature(t *testing.T) {
	body := []byte(`{"features":{"tengu_auto_mode_config":{"defaultValue":{"enabled":"opt-in","twoStageClassifier":true}}}}`)
	out, changed, err := InjectAutoMode(body)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	var resp map[string]any
	json.Unmarshal(out, &resp)
	features := resp["features"].(map[string]any)
	ac := features["tengu_auto_mode_config"].(map[string]any)
	dv := ac["defaultValue"].(map[string]any)
	if dv["enabled"] != "enabled" {
		t.Fatalf("got enabled %v", dv["enabled"])
	}
	if dv["twoStageClassifier"] != true {
		t.Fatal("existing fields should be preserved")
	}
}

func TestInjectAutoMode_NoFeatures(t *testing.T) {
	body := []byte(`{"status":"ok"}`)
	_, changed, err := InjectAutoMode(body)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected no change when no features key")
	}
}

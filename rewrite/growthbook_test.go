package rewrite

import (
	"encoding/json"
	"testing"
	"github.com/wow-look-at-my/testify/require"
)

func TestInjectAutoMode_NewFeature(t *testing.T) {
	body := []byte(`{"features":{"some_other_flag":{"defaultValue":true}}}`)
	out, changed, err := InjectAutoMode(body)
	require.Nil(t, err)

	require.True(t, changed)

	var resp map[string]any
	json.Unmarshal(out, &resp)
	features := resp["features"].(map[string]any)
	ac := features["tengu_auto_mode_config"].(map[string]any)
	dv := ac["defaultValue"].(map[string]any)
	require.Equal(t, "enabled", dv["enabled"])

	models := dv["allowModels"].([]any)
	require.False(t, len(models) != 1 || models[0] != "*")

}

func TestInjectAutoMode_ExistingFeature(t *testing.T) {
	body := []byte(`{"features":{"tengu_auto_mode_config":{"defaultValue":{"enabled":"opt-in","twoStageClassifier":true}}}}`)
	out, changed, err := InjectAutoMode(body)
	require.Nil(t, err)

	require.True(t, changed)

	var resp map[string]any
	json.Unmarshal(out, &resp)
	features := resp["features"].(map[string]any)
	ac := features["tengu_auto_mode_config"].(map[string]any)
	dv := ac["defaultValue"].(map[string]any)
	require.Equal(t, "enabled", dv["enabled"])

	require.Equal(t, true, dv["twoStageClassifier"])

}

func TestInjectAutoMode_NoFeatures(t *testing.T) {
	body := []byte(`{"status":"ok"}`)
	_, changed, err := InjectAutoMode(body)
	require.Nil(t, err)

	require.False(t, changed)

}

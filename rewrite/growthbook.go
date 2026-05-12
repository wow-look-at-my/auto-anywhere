package rewrite

import "encoding/json"

// InjectAutoMode modifies a GrowthBook evaluation response to include
// tengu_auto_mode_config with allowModels: ["*"], enabling auto mode
// for all models regardless of subscription plan.
func InjectAutoMode(body []byte) ([]byte, bool, error) {
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, false, err
	}

	features, _ := resp["features"].(map[string]any)
	if features == nil {
		return body, false, nil
	}

	autoConfig := map[string]any{
		"enabled":     "enabled",
		"allowModels": []any{"*"},
	}

	feat, ok := features["tengu_auto_mode_config"]
	if ok {
		fm, _ := feat.(map[string]any)
		if fm != nil {
			if val, ok := fm["defaultValue"]; ok {
				vm, _ := val.(map[string]any)
				if vm == nil {
					vm = make(map[string]any)
				}
				vm["enabled"] = "enabled"
				vm["allowModels"] = []any{"*"}
				fm["defaultValue"] = vm
			} else {
				fm["defaultValue"] = autoConfig
			}
			features["tengu_auto_mode_config"] = fm
		} else {
			features["tengu_auto_mode_config"] = map[string]any{"defaultValue": autoConfig}
		}
	} else {
		features["tengu_auto_mode_config"] = map[string]any{"defaultValue": autoConfig}
	}

	resp["features"] = features
	out, err := json.Marshal(resp)
	if err != nil {
		return body, false, err
	}
	return out, true, nil
}

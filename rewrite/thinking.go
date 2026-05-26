package rewrite

import (
	"encoding/json"
	"strings"
)

const defaultBudgetTokens = 10000

// InjectThinking forces thinking with summarized display on a /v1/messages
// request body. Uses adaptive thinking for 4-7 models, extended thinking
// for other thinking-capable models, and skips haiku models entirely.
func InjectThinking(body []byte) ([]byte, bool, error) {
	var msg map[string]any
	if err := json.Unmarshal(body, &msg); err != nil {
		return body, false, err
	}

	model, _ := msg["model"].(string)
	modelLower := strings.ToLower(model)

	if strings.Contains(modelLower, "haiku") {
		return body, false, nil
	}

	thinking, _ := msg["thinking"].(map[string]any)
	if thinking == nil {
		thinking = make(map[string]any)
	}

	changed := false
	supportsAdaptive := strings.Contains(modelLower, "4-7")

	if supportsAdaptive {
		if thinking["type"] != "adaptive" {
			thinking["type"] = "adaptive"
			changed = true
		}
		if _, ok := thinking["budget_tokens"]; ok {
			delete(thinking, "budget_tokens")
			changed = true
		}
	} else {
		if thinking["type"] != "enabled" {
			thinking["type"] = "enabled"
			changed = true
		}
		if _, ok := thinking["budget_tokens"]; !ok {
			thinking["budget_tokens"] = float64(defaultBudgetTokens)
			changed = true
		}
	}

	if thinking["display"] != "summarized" {
		thinking["display"] = "summarized"
		changed = true
	}

	if temp, ok := msg["temperature"]; ok {
		if t, _ := temp.(float64); t != 1 {
			msg["temperature"] = float64(1)
			changed = true
		}
	}

	if !changed {
		return body, false, nil
	}

	msg["thinking"] = thinking
	out, err := json.Marshal(msg)
	if err != nil {
		return body, false, err
	}
	return out, true, nil
}

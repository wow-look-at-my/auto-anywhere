package rewrite

import (
	"encoding/json"
	"strings"
)

const defaultBudgetTokens = 10000

// InjectThinking forces thinking with summarized display on a /v1/messages
// request body. Uses adaptive thinking for 4-7 models and extended thinking
// (type "enabled" + budget_tokens) for all other models.
// Skips injection when tool_choice forces a specific tool, since the API
// rejects thinking + forced tool_choice.
func InjectThinking(body []byte) ([]byte, bool, error) {
	var msg map[string]any
	if err := json.Unmarshal(body, &msg); err != nil {
		return body, false, err
	}

	if forcesToolChoice(msg) {
		return body, false, nil
	}

	model, _ := msg["model"].(string)

	thinking, _ := msg["thinking"].(map[string]any)
	if thinking == nil {
		thinking = make(map[string]any)
	}

	changed := false
	supportsAdaptive := strings.Contains(strings.ToLower(model), "4-7")

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

	if maxTok, ok := msg["max_tokens"].(float64); ok && maxTok > 0 {
		if bt, ok := thinking["budget_tokens"].(float64); ok && bt >= maxTok {
			thinking["budget_tokens"] = maxTok - 1
			changed = true
		}
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

// forcesToolChoice returns true when tool_choice is set to "any" or a
// specific tool (type "tool"), both of which are incompatible with thinking.
func forcesToolChoice(msg map[string]any) bool {
	tc, ok := msg["tool_choice"].(map[string]any)
	if !ok {
		return false
	}
	t, _ := tc["type"].(string)
	return t == "any" || t == "tool"
}

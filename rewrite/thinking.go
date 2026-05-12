package rewrite

import "encoding/json"

// InjectThinking forces adaptive thinking with summarized display on a
// /v1/messages request body. Returns the modified body and whether any
// change was made.
func InjectThinking(body []byte) ([]byte, bool, error) {
	var msg map[string]any
	if err := json.Unmarshal(body, &msg); err != nil {
		return body, false, err
	}

	thinking, _ := msg["thinking"].(map[string]any)
	if thinking == nil {
		thinking = make(map[string]any)
	}

	changed := false

	if thinking["type"] != "adaptive" {
		thinking["type"] = "adaptive"
		changed = true
	}
	if thinking["display"] != "summarized" {
		thinking["display"] = "summarized"
		changed = true
	}

	// Remove budget_tokens if present — adaptive thinking doesn't use it.
	if _, ok := thinking["budget_tokens"]; ok {
		delete(thinking, "budget_tokens")
		changed = true
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

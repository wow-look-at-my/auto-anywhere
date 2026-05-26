package rewrite

import (
	"encoding/json"
	"testing"

	"github.com/wow-look-at-my/testify/require"
)

func TestInjectThinking_Adaptive47(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-7","messages":[{"role":"user","content":"hi"}]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	json.Unmarshal(out, &msg)
	th := msg["thinking"].(map[string]any)
	require.Equal(t, "adaptive", th["type"])
	require.Equal(t, "summarized", th["display"])
}

func TestInjectThinking_Adaptive47AlreadySet(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-7","thinking":{"type":"adaptive","display":"summarized"},"messages":[]}`)
	_, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.False(t, changed)
}

func TestInjectThinking_Adaptive47RemovesBudget(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-7","thinking":{"type":"enabled","budget_tokens":10000},"messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	json.Unmarshal(out, &msg)
	th := msg["thinking"].(map[string]any)
	require.Equal(t, "adaptive", th["type"])
	_, ok := th["budget_tokens"]
	require.False(t, ok)
}

func TestInjectThinking_Extended46(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-6","messages":[{"role":"user","content":"hi"}]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	json.Unmarshal(out, &msg)
	th := msg["thinking"].(map[string]any)
	require.Equal(t, "enabled", th["type"])
	require.Equal(t, "summarized", th["display"])
	require.Equal(t, float64(defaultBudgetTokens), th["budget_tokens"])
}

func TestInjectThinking_Extended46PreservesBudget(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-6","thinking":{"type":"enabled","budget_tokens":20000},"messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	json.Unmarshal(out, &msg)
	th := msg["thinking"].(map[string]any)
	require.Equal(t, "enabled", th["type"])
	require.Equal(t, float64(20000), th["budget_tokens"])
	require.Equal(t, "summarized", th["display"])
}

func TestInjectThinking_Extended46AlreadySet(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-6","thinking":{"type":"enabled","budget_tokens":10000,"display":"summarized"},"messages":[]}`)
	_, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.False(t, changed)
}

func TestInjectThinking_Disabled46(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-6","thinking":{"type":"disabled"},"messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	json.Unmarshal(out, &msg)
	th := msg["thinking"].(map[string]any)
	require.Equal(t, "enabled", th["type"])
	require.Equal(t, "summarized", th["display"])
	require.Equal(t, float64(defaultBudgetTokens), th["budget_tokens"])
}

func TestInjectThinking_Disabled47(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-7","thinking":{"type":"disabled"},"messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	json.Unmarshal(out, &msg)
	th := msg["thinking"].(map[string]any)
	require.Equal(t, "adaptive", th["type"])
	require.Equal(t, "summarized", th["display"])
}

func TestInjectThinking_HaikuExtended(t *testing.T) {
	body := []byte(`{"model":"claude-haiku-4-5-20251001","messages":[{"role":"user","content":"hi"}]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	json.Unmarshal(out, &msg)
	th := msg["thinking"].(map[string]any)
	require.Equal(t, "enabled", th["type"])
	require.Equal(t, "summarized", th["display"])
	require.Equal(t, float64(defaultBudgetTokens), th["budget_tokens"])
}

func TestInjectThinking_HaikuPreservesBudget(t *testing.T) {
	body := []byte(`{"model":"claude-haiku-4-5","thinking":{"type":"enabled","budget_tokens":5000},"messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	json.Unmarshal(out, &msg)
	th := msg["thinking"].(map[string]any)
	require.Equal(t, "enabled", th["type"])
	require.Equal(t, float64(5000), th["budget_tokens"])
	require.Equal(t, "summarized", th["display"])
}

func TestInjectThinking_TemperatureOverride(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-6","temperature":0,"messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	json.Unmarshal(out, &msg)
	require.Equal(t, float64(1), msg["temperature"])
	th := msg["thinking"].(map[string]any)
	require.Equal(t, "enabled", th["type"])
}

func TestInjectThinking_TemperatureAlreadyOne(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-7","thinking":{"type":"adaptive","display":"summarized"},"temperature":1,"messages":[]}`)
	_, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.False(t, changed)
}

func TestInjectThinking_SkipsWhenToolChoiceForcesTool(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-6","tool_choice":{"type":"tool","name":"web_search"},"messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.False(t, changed)

	var msg map[string]any
	require.Nil(t, json.Unmarshal(out, &msg))
	require.Nil(t, msg["thinking"])
}

func TestInjectThinking_SkipsWhenToolChoiceAny(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-7","tool_choice":{"type":"any"},"messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.False(t, changed)

	var msg map[string]any
	require.Nil(t, json.Unmarshal(out, &msg))
	require.Nil(t, msg["thinking"])
}

func TestInjectThinking_AllowsToolChoiceAuto(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-6","tool_choice":{"type":"auto"},"messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	require.Nil(t, json.Unmarshal(out, &msg))
	th := msg["thinking"].(map[string]any)
	require.Equal(t, "enabled", th["type"])
}

func TestInjectThinking_ClampsBudgetToMaxTokens(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-6","max_tokens":4096,"messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	require.Nil(t, json.Unmarshal(out, &msg))
	th := msg["thinking"].(map[string]any)
	require.Equal(t, "enabled", th["type"])
	require.Equal(t, float64(4095), th["budget_tokens"])
}

func TestInjectThinking_ClampsBudgetPreservedByCallerToMaxTokens(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-6","max_tokens":5000,"thinking":{"type":"enabled","budget_tokens":10000},"messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	require.Nil(t, json.Unmarshal(out, &msg))
	th := msg["thinking"].(map[string]any)
	require.Equal(t, float64(4999), th["budget_tokens"])
}

func TestInjectThinking_NoClampsWhenMaxTokensLargeEnough(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-6","max_tokens":50000,"messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	require.Nil(t, json.Unmarshal(out, &msg))
	th := msg["thinking"].(map[string]any)
	require.Equal(t, float64(defaultBudgetTokens), th["budget_tokens"])
}

func TestInjectThinking_AllowsNoToolChoice(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-6","messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)
	require.True(t, changed)

	var msg map[string]any
	require.Nil(t, json.Unmarshal(out, &msg))
	th := msg["thinking"].(map[string]any)
	require.Equal(t, "enabled", th["type"])
}

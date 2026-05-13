package rewrite

import (
	"encoding/json"
	"testing"
	"github.com/wow-look-at-my/testify/require"
)

func TestInjectThinking_NoThinking(t *testing.T) {
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

func TestInjectThinking_AlreadySet(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-7","thinking":{"type":"adaptive","display":"summarized"},"messages":[]}`)
	_, changed, err := InjectThinking(body)
	require.Nil(t, err)

	require.False(t, changed)

}

func TestInjectThinking_OverrideBudget(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-6","thinking":{"type":"enabled","budget_tokens":10000},"messages":[]}`)
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

func TestInjectThinking_Disabled(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-6","thinking":{"type":"disabled"},"messages":[]}`)
	out, changed, err := InjectThinking(body)
	require.Nil(t, err)

	require.True(t, changed)

	var msg map[string]any
	json.Unmarshal(out, &msg)
	th := msg["thinking"].(map[string]any)
	require.Equal(t, "adaptive", th["type"])

	require.Equal(t, "summarized", th["display"])

}

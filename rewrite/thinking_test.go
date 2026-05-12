package rewrite

import (
	"encoding/json"
	"testing"
)

func TestInjectThinking_NoThinking(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-7","messages":[{"role":"user","content":"hi"}]}`)
	out, changed, err := InjectThinking(body)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	var msg map[string]any
	json.Unmarshal(out, &msg)
	th := msg["thinking"].(map[string]any)
	if th["type"] != "adaptive" {
		t.Fatalf("got type %v", th["type"])
	}
	if th["display"] != "summarized" {
		t.Fatalf("got display %v", th["display"])
	}
}

func TestInjectThinking_AlreadySet(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-7","thinking":{"type":"adaptive","display":"summarized"},"messages":[]}`)
	_, changed, err := InjectThinking(body)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected no change")
	}
}

func TestInjectThinking_OverrideBudget(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-6","thinking":{"type":"enabled","budget_tokens":10000},"messages":[]}`)
	out, changed, err := InjectThinking(body)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	var msg map[string]any
	json.Unmarshal(out, &msg)
	th := msg["thinking"].(map[string]any)
	if th["type"] != "adaptive" {
		t.Fatalf("got type %v", th["type"])
	}
	if _, ok := th["budget_tokens"]; ok {
		t.Fatal("budget_tokens should be removed")
	}
}

func TestInjectThinking_Disabled(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-6","thinking":{"type":"disabled"},"messages":[]}`)
	out, changed, err := InjectThinking(body)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected change")
	}
	var msg map[string]any
	json.Unmarshal(out, &msg)
	th := msg["thinking"].(map[string]any)
	if th["type"] != "adaptive" {
		t.Fatalf("got type %v", th["type"])
	}
	if th["display"] != "summarized" {
		t.Fatalf("got display %v", th["display"])
	}
}

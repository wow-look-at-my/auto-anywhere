//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/wow-look-at-my/auto-anywhere/proxy"
)

func TestThinkingByModel(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	p, err := proxy.New(proxy.Config{Upstream: "https://api.anthropic.com"})
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(p)
	defer srv.Close()

	models := []string{
		"claude-haiku-4-5-20251001",
		"claude-sonnet-4-6",
		"claude-opus-4-7",
	}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			reqBody := map[string]any{
				"model":      model,
				"max_tokens": 2048,
				"messages": []map[string]string{
					{"role": "user", "content": "Reply with only the word 'hi'."},
				},
				"thinking": map[string]any{
					"type":          "enabled",
					"budget_tokens": float64(1024),
				},
			}
			body, _ := json.Marshal(reqBody)

			req, _ := http.NewRequest("POST", srv.URL+"/v1/messages", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("x-api-key", apiKey)
			req.Header.Set("anthropic-version", "2023-06-01")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			respBody, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != 200 {
				t.Fatalf("status %d: %s", resp.StatusCode, string(respBody))
			}
			t.Logf("model=%s status=%d", model, resp.StatusCode)
		})
	}
}

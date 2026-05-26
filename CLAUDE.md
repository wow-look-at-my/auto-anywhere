# auto-anywhere

Anthropic API reverse proxy that forces thinking summaries and enables auto mode.

## Build

Build and test with `go-toolchain` (never bare `go` commands):

```sh
go-toolchain
```

Binary lands at `./build/auto-anywhere`.

## Structure

- `cmd/` -- Cobra CLI commands (one file per subcommand)
- `proxy/` -- HTTP proxy server (reverse proxy + MITM forward proxy)
- `rewrite/` -- Request/response rewrite logic (thinking injection, GrowthBook modification)
- `integration/` -- Integration tests (build-tagged `integration`, require `ANTHROPIC_API_KEY`)

## Key design decisions

- Forces thinking with `display: "summarized"` on `/v1/messages` POST, model-aware: adaptive thinking for 4-7 models, extended thinking (`type: "enabled"` + `budget_tokens`) for all other models (4-6, Haiku, etc.); also coerces `temperature` to `1` when present, since the API requires it with thinking enabled
- Intercepts GrowthBook evaluation responses (`/api/eval/*`, `/api/features/*`, `/sub/*`) to inject `tengu_auto_mode_config` with `allowModels: ["*"]`
- MITM forward proxy handles CONNECT tunneling for `api.anthropic.com` to catch GrowthBook traffic that bypasses `ANTHROPIC_BASE_URL`
- SSE streaming responses pass through unmodified (only requests are rewritten for `/v1/messages`)

## Anthropic API constraints for thinking

These constraints MUST be satisfied by the thinking injection logic in `rewrite/thinking.go`:

- `budget_tokens >= 1024` (API minimum; requests with lower values are rejected)
- `budget_tokens < max_tokens` (API constraint; budget must be strictly less than max)
- `temperature` must be `1` when thinking is enabled
- `tool_choice` of type `"any"` or `"tool"` is incompatible with thinking (skip injection)
- 4-7 models use `type: "adaptive"` with no `budget_tokens`; all others use `type: "enabled"` with `budget_tokens`

When `max_tokens` is too small to fit `budget_tokens`, the proxy bumps `max_tokens` up (to `budget_tokens + 1`) rather than clamping `budget_tokens` down, since the proxy's purpose is to force thinking on.

## Logging

`proxy/log.go` logs request/response summaries:
- Requests: model, stream flag, tool count, last message role + content (truncated)
- Streaming responses: parsed from SSE events (model, stop_reason, token usage, content)
- Non-streaming responses: parsed from JSON body; API error responses logged at WARN level
- `extractTextContent` handles both string content and content block arrays (including `tool_result` blocks with nested content)

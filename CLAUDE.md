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

## Key design decisions

- Forces thinking with `display: "summarized"` on `/v1/messages` POST, model-aware: adaptive thinking for 4-7 models, extended thinking (`type: "enabled"` + `budget_tokens`) for 4-6 models, skipped entirely for Haiku; also coerces `temperature` to `1` when present, since the API requires it with thinking enabled
- Intercepts GrowthBook evaluation responses (`/api/eval/*`, `/api/features/*`) to inject `tengu_auto_mode_config` with `allowModels: ["*"]`
- MITM forward proxy handles CONNECT tunneling for `api.anthropic.com` to catch GrowthBook traffic that bypasses `ANTHROPIC_BASE_URL`
- SSE streaming responses pass through unmodified (only requests are rewritten for `/v1/messages`)

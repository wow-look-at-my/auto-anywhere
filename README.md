# auto-anywhere

Anthropic API reverse proxy that forces thinking summaries and enables auto mode for Claude Code.

## What it does

1. **Thinking summaries** -- Forces `thinking: { type: "adaptive", display: "summarized" }` on every `/v1/messages` request. Without this, Opus 4.7 omits thinking blocks entirely, stripping the model of access to its prior chain-of-thought across turns.

2. **Auto mode for Opus 4.6** -- Intercepts GrowthBook feature flag responses and injects `tengu_auto_mode_config` with `allowModels: ["*"]`, enabling auto mode regardless of subscription plan. This only works if GrowthBook traffic is routed through the proxy (see below).

## Install

Download a binary from [releases](https://github.com/wow-look-at-my/auto-anywhere/releases/latest), or build from source:

```sh
go-toolchain
# Binary at ./build/auto-anywhere
```

## Usage

Start the proxy:

```sh
auto-anywhere
```

Then run Claude Code pointing at it:

```sh
ANTHROPIC_BASE_URL=http://localhost:18080 claude
```

### Options

```
  -p, --port int          Listen port (default 18080)
  -u, --upstream string   Upstream API URL (default "https://api.anthropic.com")
  -v, --verbose           Verbose logging
```

## How it works

The proxy sits between Claude Code and the Anthropic API. It intercepts HTTP requests and responses:

- **Requests** to `POST /v1/messages`: The thinking config is forced to `{ type: "adaptive", display: "summarized" }`, ensuring the model always has access to summarized thinking blocks.

- **Responses** from `/api/eval/*`, `/api/features/*`: GrowthBook feature evaluation responses are modified to include auto mode configuration with `allowModels: ["*"]`.

All other traffic passes through unmodified. SSE streaming responses are not buffered.

## Auto mode limitation

Claude Code's GrowthBook SDK has `apiHost` hardcoded to `https://api.anthropic.com/`, separate from `ANTHROPIC_BASE_URL`. This means GrowthBook feature flag requests bypass the proxy by default.

**Thinking summaries work out of the box** with just `ANTHROPIC_BASE_URL`.

**Auto mode** requires GrowthBook traffic to also reach the proxy. If it doesn't reach the proxy naturally, you have the alternative of using bypass permissions mode on Opus 4.6 (less secure than auto mode but functional).

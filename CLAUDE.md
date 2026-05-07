# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

go-mqtt-bench is a cross-platform MQTT v3/v5 benchmarking CLI tool, modeled after EMQX's official `emqtt-bench`. It uses Go with Eclipse Paho MQTT client and cobra CLI framework. All user-facing strings are in Chinese.

## Build and test commands

```bash
go build ./...           # Compile everything
go vet ./...             # Static analysis
make build               # Single binary (current platform)
./scripts/build.sh all   # Cross-compile all platforms
./scripts/build.sh package  # Cross-compile + create tar.gz/zip archives
```

There are no unit tests yet, so `go test ./...` is a no-op.

## Architecture

### Entry and CLI layer

`main.go` calls `cmd.Execute()`. The `cmd` package defines three cobra subcommands — `conn`, `pub`, `sub` — each registered in its own `init()`. All three share `parseCommonConfig()` defined in `cmd/conn.go` (not duplicated). Each subcommand follows the same flow:

```
parse flags → build Config → NewXxxRunner() → NewReporter() → runner.Run(ctx) → reporter.PrintFinal()
```

The `-h` shorthand for `--host` conflicts with cobra's built-in help flag. Each subcommand pre-registers `cmd.Flags().Bool("help", false, ...)` in `init()` to prevent cobra from adding its own `-h` for help.

### Config (`internal/config`)

`CommonConfig` holds parameters shared across all subcommands. `ConnConfig`, `PubConfig`, `SubConfig` embed `CommonConfig` plus subcommand-specific fields. `CommonConfig.Hosts()` splits comma-separated hosts; `CommonConfig.MQTTVersion()` maps user-facing 3/4/5 to Paho protocol bytes.

### Runner pattern (`internal/bench`)

`Runner` interface: `Run(ctx) error`, `Stats()`, `Histogram()`. `BaseRunner` provides shared deps (Collector, Histogram, RateLimiter, TLS config) and is embedded by all three runners via pointer embedding (`*BaseRunner`).

All three runners use the same lifecycle:
1. Wrap context with `util.WithSignal()` for Ctrl+C handling
2. Loop `count` times, waiting on `RateLimiter.Wait(ctx)` before each client
3. On shutdown (`goto shutdown`): stop creating clients, wait for goroutines (`wg.Wait` / `pubWg.Wait`), then disconnect all clients

**ConnRunner**: Creates clients, connects, keeps alive until ctx.Done.
**SubRunner**: Creates clients, connects, subscribes, counts messages in `OnMessage` callback. Parses ts header for latency histogram.
**PubRunner**: Pre-renders topics for all clients upfront. Each client runs a publish loop using `time.NewTicker`. Supports `--wait-before-publishing` (waits for all `ready`), random stagger delay, per-client message limit. `PubTotal` is incremented after `Publish()` returns so `PubTotal = PubSuccess + PubFailed`.

### MQTT layer (`internal/mqtt`)

`Client` wraps a `paho.Client`. Key design decisions:
- **Synchronous publish**: `Publish()` calls `token.WaitTimeout(10s)` — blocks until QoS flow completes. Effective inflight is 1 per publisher goroutine.
- **AutoReconnect is disabled** — reconnection logic is handled by the runner layer (currently not implemented beyond retry-on-connect).
- **WriteTimeout** is set to 30s on all clients.
- `RenderTopic()` handles `%i`, `%c`, `%u`, `%s` placeholders using `strings.Builder`.
- `PayloadBuilder` reuses byte buffers across publishes. Supports optional `cnt64` (uint64 counter) and `ts` (nanosecond timestamp) headers.

### Stats (`internal/stats`)

`Collector` uses `atomic.Int64` for all counters — no mutex on the hot path. `Reporter` runs a ticker loop and computes rates from snapshot deltas. **Critical**: rate calculation uses `now.Sub(r.lastTime)` (interval since last report), NOT `time.Since(r.startTime)` (total elapsed). The `Histogram` is a simple sorted-slice implementation; planned replacement is hdrhistogram-go.

### Rate limiter (`internal/bench/limiter.go`)

Two modes: `connrate` (N/sec, higher priority) and `interval` (one every N ms). Uses `time.After` + context cancellation.

## Key dependency behavior

- **Eclipse Paho v1.5.1** (`github.com/eclipse/paho.mqtt.golang`): The synchronous `Publish()` + `WaitTimeout()` model means each publish goroutine has at most 1 message in flight. For high-throughput QoS 1/2, this becomes a bottleneck — each publish waits for broker acknowledgment before sending the next.
- **cobra v1.10.2**: `InitDefaultHelpFlag()` auto-adds `-h` shorthand for `--help` on every command. Any flag using `-h` shorthand must pre-register a `help` flag without shorthand to prevent the conflict.

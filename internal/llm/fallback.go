package llm

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// FallbackClient tries the local LLM first, and on error/timeoutexplicitly
// falls back to a cloud client (OpenRouter). This makes the Agent resilient
// to a hung/slow local llama-server: parallel Agent requests that would block
// on a single local slot instead run on OpenRouter (which is parallel-native).
//
// ponytail: per-request fallback, no caching of "local is dead" — cheap enough
// and avoids stale state if the local server recovers. If you see too many
// fallbacks, bump localTimeout or mark local dead after N failures.
type FallbackClient struct {
	local      Client
	cloud      Client
	localTry   time.Duration // give local this long before bailing
}

// NewFallbackClient wraps a local and a cloud client. localTry caps how long
// the local LLM gets before the cloud client is used.
func NewFallbackClient(local, cloud Client, localTry time.Duration) *FallbackClient {
	if localTry <= 0 {
		localTry = 20 * time.Second
	}
	return &FallbackClient{local: local, cloud: cloud, localTry: localTry}
}

func (f *FallbackClient) Name() string { return "fallback(" + f.local.Name() + "→" + f.cloud.Name() + ")" }

func (f *FallbackClient) Complete(ctx context.Context, prompt string) (string, error) {
	lctx, cancel := context.WithTimeout(ctx, f.localTry)
	defer cancel()
	out, err := f.local.Complete(lctx, prompt)
	if err == nil {
		return out, nil
	}
	// local failed/timeout → cloud
	slog.Warn("llm local failed, falling back to cloud", "error", err)
	return f.cloud.Complete(ctx, prompt)
}

// Stream is optional; if the cloud client supports it, use cloud directly
// (local streaming usually unavailable for our purposes).
func (f *FallbackClient) Stream(ctx context.Context, prompt string) (<-chan string, error) {
	if s, ok := f.cloud.(Streamer); ok {
		return s.Stream(ctx, prompt)
	}
	return nil, fmt.Errorf("streaming not supported by fallback client")
}

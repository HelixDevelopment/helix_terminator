// Package llmclient adapts the vasic-digital/LLMProvider generic OpenAI-compatible
// provider (submodules/llmprovider, module digital.vasic.llmprovider) to the narrow
// completion contract ai-service's handler package depends on (handler.LLMClient).
//
// Real construction (NewGenericClient) always targets the local HelixLLM llama.cpp
// server — the cloud tier (OpenAI/Anthropic API keys) is OPERATOR-BLOCKED and
// deliberately NOT wired here; see docs/CONTINUATION.md for the tracked item.
package llmclient

import (
	"context"
	"fmt"

	"digital.vasic.llmprovider/pkg/models"
	"digital.vasic.llmprovider/pkg/providers/generic"
)

// GenericClient adapts the llmprovider generic OpenAI-compatible provider to the
// ai-service handler's LLMClient contract.
type GenericClient struct {
	provider *generic.Provider
}

// NewGenericClient builds a GenericClient backed by the real llmprovider generic
// adapter.
//
//   - name: provider identifier for logging/metadata (e.g. "helixllm-local").
//   - apiKey: MUST be non-empty. llama-server does not enforce auth, but an empty
//     apiKey fails the adapter's own ValidateConfig contract — pass any non-empty
//     placeholder (e.g. "local-no-auth-required").
//   - baseURL: MUST be the FULL chat-completions path (e.g.
//     "http://127.0.0.1:18434/v1/chat/completions"). generic.Provider uses baseURL
//     AS-IS and appends nothing — passing a bare host silently 404s.
//   - model: default model ID used when a request does not specify one.
func NewGenericClient(name, apiKey, baseURL, model string) *GenericClient {
	return &GenericClient{provider: generic.NewGenericProvider(name, apiKey, baseURL, model)}
}

// Complete sends prompt as a single user-role message to the configured provider and
// returns the completion content and total token usage. An empty model / non-positive
// maxTokens / zero temperature fall back to the provider's configured defaults (see
// generic.Provider's request conversion — empty model keeps the provider's own default
// model, non-positive maxTokens becomes generic.DefaultMaxTokens).
func (g *GenericClient) Complete(ctx context.Context, model string, maxTokens int, temperature float64, prompt string) (content string, tokensUsed int, err error) {
	if g == nil || g.provider == nil {
		return "", 0, fmt.Errorf("llmclient: provider not configured")
	}

	req := &models.LLMRequest{
		Messages: []models.Message{{Role: "user", Content: prompt}},
		ModelParams: models.ModelParameters{
			Model:       model,
			MaxTokens:   maxTokens,
			Temperature: temperature,
		},
	}

	resp, err := g.provider.Complete(ctx, req)
	if err != nil {
		return "", 0, fmt.Errorf("llmclient: completion failed: %w", err)
	}
	if resp == nil {
		return "", 0, fmt.Errorf("llmclient: provider returned nil response")
	}
	return resp.Content, resp.TokensUsed, nil
}

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/akrishnanDG/glue-to-ccsr/pkg/config"
	"golang.org/x/time/rate"
)

// Provider is the interface for LLM providers
type Provider interface {
	Complete(ctx context.Context, prompt string) (string, float64, error)
}

// NewProvider creates a new LLM provider based on configuration
func NewProvider(cfg *config.Config) (Provider, error) {
	// Create rate limiter
	limiter := rate.NewLimiter(rate.Limit(cfg.LLM.RateLimit), 1)
	inputCost := cfg.LLM.InputTokenCost
	outputCost := cfg.LLM.OutputTokenCost

	switch cfg.LLM.Provider {
	case "openai":
		return NewOpenAIProvider(cfg.LLM.APIKey, cfg.LLM.Model, limiter, inputCost, outputCost), nil
	case "anthropic":
		return NewAnthropicProvider(cfg.LLM.APIKey, cfg.LLM.Model, limiter, inputCost, outputCost), nil
	case "ollama":
		return NewOllamaProvider(cfg.LLM.BaseURL, cfg.LLM.Model, limiter), nil
	case "local":
		return NewGenericProvider(cfg.LLM.BaseURL, cfg.LLM.Model, cfg.LLM.APIKey, limiter), nil
	case "bedrock":
		return NewBedrockProvider(cfg.LLM.Model, limiter)
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.LLM.Provider)
	}
}

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	apiKey          string
	model           string
	limiter         *rate.Limiter
	client          *http.Client
	inputTokenCost  float64
	outputTokenCost float64
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey, model string, limiter *rate.Limiter, inputCost, outputCost float64) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:          apiKey,
		model:           model,
		limiter:         limiter,
		client:          &http.Client{Timeout: 60 * time.Second},
		inputTokenCost:  inputCost,
		outputTokenCost: outputCost,
	}
}

func (p *OpenAIProvider) Complete(ctx context.Context, prompt string) (string, float64, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return "", 0, err
	}

	reqBody := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  500,
		"temperature": 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("OpenAI API error: %s", string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", 0, err
	}

	if len(result.Choices) == 0 {
		return "", 0, fmt.Errorf("no response from OpenAI")
	}

	cost := float64(result.Usage.PromptTokens)*p.inputTokenCost + float64(result.Usage.CompletionTokens)*p.outputTokenCost

	return result.Choices[0].Message.Content, cost, nil
}

// AnthropicProvider implements the Provider interface for Anthropic
type AnthropicProvider struct {
	apiKey          string
	model           string
	limiter         *rate.Limiter
	client          *http.Client
	inputTokenCost  float64
	outputTokenCost float64
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey, model string, limiter *rate.Limiter, inputCost, outputCost float64) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey:          apiKey,
		model:           model,
		limiter:         limiter,
		client:          &http.Client{Timeout: 60 * time.Second},
		inputTokenCost:  inputCost,
		outputTokenCost: outputCost,
	}
}

func (p *AnthropicProvider) Complete(ctx context.Context, prompt string) (string, float64, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return "", 0, err
	}

	reqBody := map[string]interface{}{
		"model":      p.model,
		"max_tokens": 500,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("Anthropic API error: %s", string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", 0, err
	}

	if len(result.Content) == 0 {
		return "", 0, fmt.Errorf("no response from Anthropic")
	}

	cost := float64(result.Usage.InputTokens)*p.inputTokenCost + float64(result.Usage.OutputTokens)*p.outputTokenCost

	return result.Content[0].Text, cost, nil
}

// OllamaProvider implements the Provider interface for Ollama (local)
type OllamaProvider struct {
	baseURL string
	model   string
	limiter *rate.Limiter
	client  *http.Client
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(baseURL, model string, limiter *rate.Limiter) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		limiter: limiter,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *OllamaProvider) Complete(ctx context.Context, prompt string) (string, float64, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return "", 0, err
	}

	reqBody := map[string]interface{}{
		"model":  p.model,
		"prompt": prompt,
		"stream": false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("Ollama API error: %s", string(respBody))
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", 0, err
	}

	// Local LLM has no cost
	return result.Response, 0, nil
}

// GenericProvider implements the Provider interface for generic OpenAI-compatible APIs
type GenericProvider struct {
	baseURL string
	model   string
	apiKey  string
	limiter *rate.Limiter
	client  *http.Client
}

// NewGenericProvider creates a new generic provider
func NewGenericProvider(baseURL, model, apiKey string, limiter *rate.Limiter) *GenericProvider {
	return &GenericProvider{
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		limiter: limiter,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *GenericProvider) Complete(ctx context.Context, prompt string) (string, float64, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return "", 0, err
	}

	reqBody := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  500,
		"temperature": 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("API error: %s", string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", 0, err
	}

	if len(result.Choices) == 0 {
		return "", 0, fmt.Errorf("no response from API")
	}

	// Local LLM has no cost
	return result.Choices[0].Message.Content, 0, nil
}

// BedrockProvider implements the Provider interface for AWS Bedrock
type BedrockProvider struct {
	model   string
	limiter *rate.Limiter
}

// NewBedrockProvider creates a new Bedrock provider
func NewBedrockProvider(model string, limiter *rate.Limiter) (*BedrockProvider, error) {
	return &BedrockProvider{
		model:   model,
		limiter: limiter,
	}, nil
}

func (p *BedrockProvider) Complete(ctx context.Context, prompt string) (string, float64, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return "", 0, err
	}

	// Note: Full Bedrock implementation would require AWS SDK
	// This is a placeholder that shows the structure
	return "", 0, fmt.Errorf("Bedrock provider requires AWS SDK implementation")
}

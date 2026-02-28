package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cool-code-cleanup/internal/cleanup"
	"cool-code-cleanup/internal/config"
	"cool-code-cleanup/internal/rules"
)

type OpenAIExecutor struct {
	apiKey string
	model  string
	client *http.Client
}

func NewOpenAIExecutorFromConfig(cfg config.Config) (*OpenAIExecutor, error) {
	apiKey := strings.TrimSpace(cfg.OpenAI.APIKeyValue)
	if apiKey == "" {
		envName := strings.TrimSpace(cfg.OpenAI.APIKeyEnv)
		if envName == "" {
			envName = "OPENAI_API_KEY"
		}
		apiKey = strings.TrimSpace(os.Getenv(envName))
	}
	if apiKey == "" {
		return nil, fmt.Errorf("missing OpenAI API key; set %s or configure openai.api_key_value", cfg.OpenAI.APIKeyEnv)
	}
	model := strings.TrimSpace(cfg.OpenAI.Model)
	if model == "" {
		model = "gpt-5"
	}
	return &OpenAIExecutor{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 45 * time.Second},
	}, nil
}

type chatCompletionRequest struct {
	Model          string              `json:"model"`
	Messages       []map[string]string `json:"messages"`
	ResponseFormat map[string]string   `json:"response_format,omitempty"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type cleanupLLMOutput struct {
	Changed        bool   `json:"changed"`
	Summary        string `json:"summary"`
	UpdatedContent string `json:"updated_content"`
}

func (e *OpenAIExecutor) TransformFile(ctx context.Context, filePath, content string, selectedRules []rules.Rule, safe, aggressive bool) (cleanup.TransformResult, error) {
	ruleJSON, _ := json.Marshal(selectedRules)
	safety := "safe=true aggressive=false"
	if aggressive {
		safety = "safe=true aggressive=true"
	}
	if !safe {
		safety = "safe=false aggressive=true"
	}

	system := "You are a code cleanup engine. Apply only the selected rules to one file. Return strict JSON with keys: changed, summary, updated_content."
	user := fmt.Sprintf(
		"File path: %s\nSafety mode: %s\nSelected rules (json): %s\nOriginal content:\n%s\n\nReturn JSON only.",
		filePath, safety, string(ruleJSON), content,
	)

	reqBody := chatCompletionRequest{
		Model: e.model,
		Messages: []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		ResponseFormat: map[string]string{"type": "json_object"},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return cleanup.TransformResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return cleanup.TransformResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return cleanup.TransformResult{}, err
	}
	defer resp.Body.Close()

	var parsed chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return cleanup.TransformResult{}, fmt.Errorf("decode OpenAI response: %w", err)
	}
	if parsed.Error != nil {
		return cleanup.TransformResult{}, fmt.Errorf("OpenAI API error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return cleanup.TransformResult{}, fmt.Errorf("OpenAI returned no choices")
	}

	text := strings.TrimSpace(parsed.Choices[0].Message.Content)
	var out cleanupLLMOutput
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return cleanup.TransformResult{}, fmt.Errorf("parse cleanup JSON output: %w", err)
	}
	result := cleanup.TransformResult{
		Changed: out.Changed,
		Summary: strings.TrimSpace(out.Summary),
		Content: out.UpdatedContent,
	}
	if result.Content == "" {
		result.Content = content
	}
	if !result.Changed {
		result.Content = content
	}
	return result, nil
}

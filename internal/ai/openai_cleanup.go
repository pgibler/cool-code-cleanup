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
		client: &http.Client{Timeout: 90 * time.Second},
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

type cleanupProjectLLMOutput struct {
	Changed bool   `json:"changed"`
	Summary string `json:"summary"`
	Files   []struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	} `json:"files"`
}

func (e *OpenAIExecutor) TransformProject(ctx context.Context, _ string, files []cleanup.ProjectFile, task cleanup.Task, selectedRules []rules.Rule, safe, aggressive bool) (cleanup.ProjectTransformResult, error) {
	ruleJSON, _ := json.Marshal(selectedRules)
	taskJSON, _ := json.Marshal(task)
	filesJSON, _ := json.Marshal(files)

	safety := "safe=true aggressive=false"
	if aggressive {
		safety = "safe=true aggressive=true"
	}
	if !safe {
		safety = "safe=false aggressive=true"
	}

	system := "You are a code cleanup engine. Execute one cleanup task across multiple files. Return strict JSON with keys: changed, summary, files. files is an array of {path, content} for modified files only."
	user := fmt.Sprintf(
		"Safety mode: %s\nTask (json): %s\nSelected rules (json): %s\nFiles in task scope (json): %s\n\nApply only task-relevant changes. Return JSON only.",
		safety, string(taskJSON), string(ruleJSON), string(filesJSON),
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
		return cleanup.ProjectTransformResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return cleanup.ProjectTransformResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return cleanup.ProjectTransformResult{}, err
	}
	defer resp.Body.Close()

	var parsed chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return cleanup.ProjectTransformResult{}, fmt.Errorf("decode OpenAI response: %w", err)
	}
	if parsed.Error != nil {
		return cleanup.ProjectTransformResult{}, fmt.Errorf("OpenAI API error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return cleanup.ProjectTransformResult{}, fmt.Errorf("OpenAI returned no choices")
	}

	text := strings.TrimSpace(parsed.Choices[0].Message.Content)
	var out cleanupProjectLLMOutput
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return cleanup.ProjectTransformResult{}, fmt.Errorf("parse cleanup JSON output: %w", err)
	}

	changedFiles := map[string]string{}
	for _, f := range out.Files {
		p := strings.TrimSpace(f.Path)
		if p == "" {
			continue
		}
		changedFiles[p] = f.Content
	}

	return cleanup.ProjectTransformResult{
		Changed:      out.Changed && len(changedFiles) > 0,
		Summary:      strings.TrimSpace(out.Summary),
		ChangedFiles: changedFiles,
	}, nil
}

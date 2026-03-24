package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

const (
	defaultHTTPTimeout       = 6 * time.Minute
	perBatchRequestTimeout   = 4 * time.Minute
	initialBatchMaxBytes     = 90_000
	maxBatchErrorSummarySize = 220
)

func NewOpenAIExecutorFromConfig(cfg config.Config) (*OpenAIExecutor, error) {
	apiKey := strings.TrimSpace(cfg.OpenAI.APIKeyValue)
	envName := strings.TrimSpace(cfg.OpenAI.APIKeyEnv)
	if envName == "" {
		envName = "OPENAI_API_KEY"
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv(envName))
	}
	if apiKey == "" {
		return nil, fmt.Errorf("missing OpenAI API key; set %s or configure openai.api_key_value", envName)
	}
	model := strings.TrimSpace(cfg.OpenAI.Model)
	if model == "" {
		model = "gpt-5"
	}
	return &OpenAIExecutor{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: defaultHTTPTimeout},
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
	if len(files) == 0 {
		return cleanup.ProjectTransformResult{Changed: false, ChangedFiles: map[string]string{}}, nil
	}
	batches := batchFiles(files, initialBatchMaxBytes)
	changedFiles := map[string]string{}
	summaries := make([]string, 0, len(batches))
	failedBatches := 0
	var lastErr error
	for _, batch := range batches {
		res, err := e.transformBatchAdaptive(ctx, batch, task, selectedRules, safe, aggressive)
		if err != nil {
			failedBatches++
			lastErr = err
			summaries = append(summaries, "batch failed: "+shortError(err.Error(), maxBatchErrorSummarySize))
			continue
		}
		for p, c := range res.ChangedFiles {
			changedFiles[p] = c
		}
		if strings.TrimSpace(res.Summary) != "" {
			summaries = append(summaries, res.Summary)
		}
	}
	if failedBatches == len(batches) {
		if lastErr == nil {
			lastErr = fmt.Errorf("all cleanup batches failed")
		}
		return cleanup.ProjectTransformResult{}, lastErr
	}
	return cleanup.ProjectTransformResult{
		Changed:      len(changedFiles) > 0,
		Summary:      strings.Join(summaries, "; "),
		ChangedFiles: changedFiles,
	}, nil
}

func (e *OpenAIExecutor) transformBatchAdaptive(ctx context.Context, files []cleanup.ProjectFile, task cleanup.Task, selectedRules []rules.Rule, safe, aggressive bool) (cleanup.ProjectTransformResult, error) {
	if len(files) == 0 {
		return cleanup.ProjectTransformResult{Changed: false, ChangedFiles: map[string]string{}}, nil
	}
	reqCtx, cancel := context.WithTimeout(ctx, perBatchRequestTimeout)
	res, err := e.transformBatch(reqCtx, files, task, selectedRules, safe, aggressive)
	cancel()
	if err == nil {
		return res, nil
	}
	if !isRetryable(err) || len(files) == 1 {
		return cleanup.ProjectTransformResult{}, err
	}

	mid := len(files) / 2
	left, errLeft := e.transformBatchAdaptive(ctx, files[:mid], task, selectedRules, safe, aggressive)
	right, errRight := e.transformBatchAdaptive(ctx, files[mid:], task, selectedRules, safe, aggressive)
	switch {
	case errLeft != nil && errRight != nil:
		return cleanup.ProjectTransformResult{}, fmt.Errorf("adaptive split failed: left=%s right=%s", shortError(errLeft.Error(), 120), shortError(errRight.Error(), 120))
	case errLeft != nil:
		summary := strings.TrimSpace(right.Summary)
		if summary == "" {
			summary = "partial success"
		}
		right.Summary = summary + "; partial failure: " + shortError(errLeft.Error(), 120)
		return right, nil
	case errRight != nil:
		summary := strings.TrimSpace(left.Summary)
		if summary == "" {
			summary = "partial success"
		}
		left.Summary = summary + "; partial failure: " + shortError(errRight.Error(), 120)
		return left, nil
	default:
		merged := cleanup.ProjectTransformResult{
			Changed:      left.Changed || right.Changed,
			Summary:      joinNonEmpty(left.Summary, right.Summary, "; "),
			ChangedFiles: map[string]string{},
		}
		for p, c := range left.ChangedFiles {
			merged.ChangedFiles[p] = c
		}
		for p, c := range right.ChangedFiles {
			merged.ChangedFiles[p] = c
		}
		return merged, nil
	}
}

func (e *OpenAIExecutor) transformBatch(ctx context.Context, files []cleanup.ProjectFile, task cleanup.Task, selectedRules []rules.Rule, safe, aggressive bool) (cleanup.ProjectTransformResult, error) {
	ruleJSON, err := json.Marshal(selectedRules)
	if err != nil {
		return cleanup.ProjectTransformResult{}, fmt.Errorf("marshal selected rules: %w", err)
	}
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return cleanup.ProjectTransformResult{}, fmt.Errorf("marshal task: %w", err)
	}
	filesJSON, err := json.Marshal(files)
	if err != nil {
		return cleanup.ProjectTransformResult{}, fmt.Errorf("marshal files: %w", err)
	}

	var safety string
	switch {
	case !safe:
		safety = "safe=false aggressive=true"
	case aggressive:
		safety = "safe=true aggressive=true"
	default:
		safety = "safe=true aggressive=false"
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
		return cleanup.ProjectTransformResult{}, fmt.Errorf("marshal OpenAI request: %w", err)
	}
	text, err := e.chatCompletionsWithRetry(ctx, body, 3)
	if err != nil {
		return cleanup.ProjectTransformResult{}, err
	}
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
		Changed:      len(changedFiles) > 0,
		Summary:      strings.TrimSpace(out.Summary),
		ChangedFiles: changedFiles,
	}, nil
}

func (e *OpenAIExecutor) chatCompletionsWithRetry(ctx context.Context, body []byte, maxAttempts int) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("build OpenAI request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := e.client.Do(req)
		if err != nil {
			lastErr = err
			if !isRetryable(err) || attempt == maxAttempts {
				return "", fmt.Errorf("openai request failed: %w", err)
			}
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
			continue
		}

		// Ensure body is always closed
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			b, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			msg := strings.TrimSpace(string(b))
			if msg == "" {
				msg = http.StatusText(resp.StatusCode)
			}
			if len(msg) > 300 {
				msg = msg[:300] + "..."
			}
			lastErr = fmt.Errorf("OpenAI HTTP %d: %s", resp.StatusCode, msg)
			// Retry on 5xx, otherwise stop
			if attempt == maxAttempts || resp.StatusCode < 500 {
				return "", lastErr
			}
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
			continue
		}

		var parsed chatCompletionResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&parsed)
		_ = resp.Body.Close()
		if decodeErr != nil {
			lastErr = fmt.Errorf("decode OpenAI response: %w", decodeErr)
			if attempt == maxAttempts {
				return "", lastErr
			}
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
			continue
		}
		if parsed.Error != nil {
			lastErr = fmt.Errorf("OpenAI API error: %s", parsed.Error.Message)
			if attempt == maxAttempts {
				return "", lastErr
			}
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
			continue
		}
		if len(parsed.Choices) == 0 {
			lastErr = fmt.Errorf("OpenAI returned no choices")
			if attempt == maxAttempts {
				return "", lastErr
			}
			time.Sleep(time.Duration(attempt*attempt) * time.Second)
			continue
		}
		return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
	}
	return "", lastErr
}

func isRetryable(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") || strings.Contains(msg, "temporarily unavailable")
}

func batchFiles(files []cleanup.ProjectFile, maxBytes int) [][]cleanup.ProjectFile {
	var batches [][]cleanup.ProjectFile
	var cur []cleanup.ProjectFile
	curSize := 0
	for _, f := range files {
		size := len(f.Path) + len(f.Content)
		if len(cur) > 0 && curSize+size > maxBytes {
			batches = append(batches, cur)
			cur = nil
			curSize = 0
		}
		cur = append(cur, f)
		curSize += size
	}
	if len(cur) > 0 {
		batches = append(batches, cur)
	}
	return batches
}

func shortError(msg string, max int) string {
	msg = strings.TrimSpace(msg)
	if max <= 0 || len(msg) <= max {
		return msg
	}
	return msg[:max] + "..."
}

func joinNonEmpty(left, right, sep string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	switch {
	case left == "":
		return right
	case right == "":
		return left
	default:
		return left + sep + right
	}
}

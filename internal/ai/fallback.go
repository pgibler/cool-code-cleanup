package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cool-code-cleanup/internal/config"
	"cool-code-cleanup/internal/dependency"
	"cool-code-cleanup/internal/discovery"
)

type NopFallback struct{}

func (NopFallback) Infer(_ []discovery.Route) (dependency.Graph, error) {
	return dependency.Graph{
		Dependencies: map[string][]string{},
		Confidence:   "low",
		Rationale:    "fallback unavailable; no inferred dependencies",
	}, nil
}

type RouteFallback interface {
	InferRoutes(projectRoot string, known []discovery.Route) ([]discovery.Route, error)
}

type OpenAIFallback struct {
	executor *OpenAIExecutor
}

func NewOpenAIFallbackFromConfig(cfg config.Config) (*OpenAIFallback, error) {
	exec, err := NewOpenAIExecutorFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &OpenAIFallback{executor: exec}, nil
}

func (f *OpenAIFallback) Infer(routes []discovery.Route) (dependency.Graph, error) {
	if f == nil || f.executor == nil || len(routes) == 0 {
		return NopFallback{}.Infer(routes)
	}

	payload, err := json.Marshal(routes)
	if err != nil {
		return NopFallback{}.Infer(routes)
	}

	system := "You infer route dependency graphs. Return strict JSON only."
	user := fmt.Sprintf(
		`Given routes (json), infer route dependencies where one route likely requires another route to run first (authentication/session/token/bootstrap dependencies).
Return strict JSON in this shape:
{"dependencies":{"<route_id>":["<dependency_route_id>"]},"rationale":"..."}
Use only route IDs that already exist in the provided routes list.

routes:
%s`, string(payload))

	body, err := json.Marshal(chatCompletionRequest{
		Model: f.executor.model,
		Messages: []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		ResponseFormat: map[string]string{"type": "json_object"},
	})
	if err != nil {
		return NopFallback{}.Infer(routes)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	text, err := f.executor.chatCompletionsWithRetry(ctx, body, 2)
	if err != nil {
		return NopFallback{}.Infer(routes)
	}

	var out struct {
		Dependencies map[string][]string `json:"dependencies"`
		Rationale    string              `json:"rationale"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return NopFallback{}.Infer(routes)
	}

	known := map[string]bool{}
	for _, r := range routes {
		known[r.ID] = true
	}

	filtered := map[string][]string{}
	for routeID, deps := range out.Dependencies {
		if !known[routeID] {
			continue
		}
		for _, depID := range deps {
			if depID == routeID || !known[depID] {
				continue
			}
			filtered[routeID] = appendUnique(filtered[routeID], depID)
		}
	}

	rationale := strings.TrimSpace(out.Rationale)
	if rationale == "" {
		rationale = "ai fallback inference"
	}
	return dependency.Graph{
		Dependencies: filtered,
		Confidence:   "low",
		Rationale:    rationale,
	}, nil
}

func (f *OpenAIFallback) InferRoutes(projectRoot string, known []discovery.Route) ([]discovery.Route, error) {
	if f == nil || f.executor == nil {
		return nil, nil
	}

	sources, err := collectSourceSnippets(projectRoot)
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, nil
	}

	knownJSON, err := json.Marshal(known)
	if err != nil {
		return nil, err
	}
	sourceJSON, err := json.Marshal(sources)
	if err != nil {
		return nil, err
	}

	system := "You infer API routes from source code snippets. Return strict JSON only."
	user := fmt.Sprintf(
		`Infer additional routes that may have been missed by deterministic parsing.
Focus on HTTP route handlers and dependency/helper routes used by authentication/session/token flows.
Return strict JSON in this shape:
{"routes":[{"method":"GET|POST|PUT|PATCH|DELETE|ANY","path":"/path","file":"relative/or/absolute/path","handler":"name","framework":"node|go|django|unknown","middleware":["..."]}]}
Only include likely real routes. Avoid duplicates already present.

Known routes (json):
%s

Source snippets (json):
%s`, string(knownJSON), string(sourceJSON))

	body, err := json.Marshal(chatCompletionRequest{
		Model: f.executor.model,
		Messages: []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		ResponseFormat: map[string]string{"type": "json_object"},
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	text, err := f.executor.chatCompletionsWithRetry(ctx, body, 2)
	if err != nil {
		return nil, err
	}

	var out struct {
		Routes []discovery.Route `json:"routes"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, err
	}

	for i := range out.Routes {
		r := &out.Routes[i]
		r.Method = normalizeMethod(r.Method)
		r.Path = normalizePath(r.Path)
		if strings.TrimSpace(r.Handler) == "" {
			r.Handler = "ai_inferred_handler"
		}
		if strings.TrimSpace(r.Framework) == "" {
			r.Framework = "unknown"
		}
		if strings.TrimSpace(r.ID) == "" {
			r.ID = inferredRouteID(*r, i)
		}
	}

	return out.Routes, nil
}

type sourceSnippet struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func collectSourceSnippets(projectRoot string) ([]sourceSnippet, error) {
	const (
		maxFiles      = 48
		maxFileBytes  = 2200
		maxTotalBytes = 52_000
	)
	out := make([]sourceSnippet, 0, maxFiles)
	total := 0

	err := filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".ccc", "node_modules", "vendor", "dist", "build", "bin":
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".js", ".ts", ".go", ".py":
		default:
			return nil
		}
		if len(out) >= maxFiles || total >= maxTotalBytes {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if len(data) > maxFileBytes {
			data = data[:maxFileBytes]
		}
		rel, relErr := filepath.Rel(projectRoot, path)
		if relErr != nil {
			rel = path
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			return nil
		}
		out = append(out, sourceSnippet{Path: filepath.ToSlash(rel), Content: content})
		total += len(content)
		return nil
	})
	return out, err
}

func inferredRouteID(r discovery.Route, idx int) string {
	file := strings.TrimSpace(r.File)
	if file == "" {
		file = "unknown"
	}
	file = filepath.Base(file)
	return "ai:" + strings.ToLower(normalizeMethod(r.Method)) + ":" + normalizePath(r.Path) + ":" + file + ":" + strconv.Itoa(idx+1)
}

func normalizeMethod(method string) string {
	m := strings.ToUpper(strings.TrimSpace(method))
	switch m {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "ANY":
		return m
	default:
		return "ANY"
	}
}

func normalizePath(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func appendUnique(list []string, item string) []string {
	for _, existing := range list {
		if existing == item {
			return list
		}
	}
	return append(list, item)
}

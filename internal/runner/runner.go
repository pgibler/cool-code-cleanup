package runner

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"cool-code-cleanup/internal/discovery"
	"cool-code-cleanup/internal/profile"
)

type Invocation struct {
	RouteID    string            `json:"route_id"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Parameters map[string]string `json:"parameters"`
	Success    bool              `json:"success"`
	Status     int               `json:"status"`
	Error      string            `json:"error,omitempty"`
}

type AppProcess struct {
	cmd *exec.Cmd
}

func (p *AppProcess) Stop() {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return
	}
	_ = p.cmd.Process.Kill()
	_, _ = p.cmd.Process.Wait()
}

func Start(projectRoot string) (*AppProcess, string) {
	// Minimal heuristic startup command selection.
	type candidate struct {
		path string
		cmd  []string
	}
	candidates := []candidate{
		{path: filepath.Join(projectRoot, "package.json"), cmd: []string{"npm", "run", "dev"}},
		{path: filepath.Join(projectRoot, "main.go"), cmd: []string{"go", "run", "."}},
		{path: filepath.Join(projectRoot, "manage.py"), cmd: []string{"python", "manage.py", "runserver", "127.0.0.1:8000"}},
	}

	for _, c := range candidates {
		if _, err := os.Stat(c.path); err == nil {
			if len(c.cmd) > 1 {
				cmd := exec.Command(c.cmd[0], c.cmd[1:]...)
				cmd.Dir = projectRoot
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Start(); err == nil {
					return &AppProcess{cmd: cmd}, strings.Join(c.cmd, " ")
				}
			}
		}
	}
	return nil, ""
}

func WaitForHealth(baseURL string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	client := &http.Client{Timeout: 2 * time.Second}
	for {
		select {
		case <-ctx.Done():
			return false
		case <-t.C:
			req, _ := http.NewRequest(http.MethodGet, baseURL, nil)
			resp, err := client.Do(req)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 500 {
					return true
				}
			}
		}
	}
}

func Execute(baseURL string, routes []discovery.Route, plans []profile.ParameterPlan, dependencies map[string][]string) []Invocation {
	routeByID := map[string]discovery.Route{}
	for _, r := range routes {
		routeByID[r.ID] = r
	}
	planByID := map[string]profile.ParameterPlan{}
	for _, p := range plans {
		planByID[p.RouteID] = p
	}

	var order []string
	seen := map[string]bool{}
	for _, r := range routes {
		for _, d := range dependencies[r.ID] {
			if !seen[d] {
				order = append(order, d)
				seen[d] = true
			}
		}
		if !seen[r.ID] {
			order = append(order, r.ID)
			seen[r.ID] = true
		}
	}

	client := &http.Client{Timeout: 5 * time.Second}
	var out []Invocation
	for _, id := range order {
		r, ok := routeByID[id]
		if !ok {
			continue
		}
		p := planByID[id]
		valid := map[string]string{}
		if len(p.Valid) > 0 {
			valid = p.Valid[0]
		}
		method := r.Method
		if method == "ANY" {
			method = http.MethodGet
		}
		url := strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(r.Path, "/")
		req, _ := http.NewRequest(method, url, nil)
		resp, err := client.Do(req)
		inv := Invocation{
			RouteID:    r.ID,
			Method:     method,
			Path:       r.Path,
			Parameters: valid,
		}
		if err != nil {
			inv.Error = err.Error()
			inv.Success = false
			out = append(out, inv)
			continue
		}
		inv.Status = resp.StatusCode
		inv.Success = resp.StatusCode >= 200 && resp.StatusCode < 400
		_ = resp.Body.Close()
		out = append(out, inv)
	}
	return out
}

func FormatInvocation(inv Invocation) string {
	check := "x"
	if inv.Success {
		check = "✓"
	}
	return fmt.Sprintf("%s %s %s params=%v status=%d", check, inv.Method, inv.Path, inv.Parameters, inv.Status)
}

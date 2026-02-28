package gitflow

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Result struct {
	Offered bool   `json:"offered"`
	Applied bool   `json:"applied"`
	Branch  string `json:"branch,omitempty"`
	Commit  string `json:"commit,omitempty"`
	Error   string `json:"error,omitempty"`
}

func CreateBranchAndCommit(mode string) Result {
	branch := fmt.Sprintf("ccc/%s-%s", mode, time.Now().UTC().Format("20060102-150405"))
	res := Result{Offered: true, Branch: branch}
	if err := run("git", "rev-parse", "--is-inside-work-tree"); err != nil {
		res.Error = "not a git repository"
		return res
	}
	if err := run("git", "checkout", "-b", branch); err != nil {
		res.Error = err.Error()
		return res
	}
	if err := run("git", "add", "-A"); err != nil {
		res.Error = err.Error()
		return res
	}
	msg := fmt.Sprintf("ccc: apply %s changes", mode)
	if err := run("git", "commit", "-m", msg); err != nil {
		res.Error = err.Error()
		return res
	}
	hash, err := output("git", "rev-parse", "HEAD")
	if err != nil {
		res.Error = err.Error()
		return res
	}
	res.Commit = strings.TrimSpace(hash)
	res.Applied = true
	return res
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v: %s", name, args, strings.TrimSpace(string(out)))
	}
	return nil
}

func output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %v: %s", name, args, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

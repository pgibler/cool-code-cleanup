package tui

import (
	"fmt"
	"strings"
)

type Action struct {
	Key      string
	Label    string
	Selected bool
}

type StepScreen struct {
	Mode        string
	StepName    string
	Description string
	Content     []string
	Actions     []Action
	InlineError string
}

func (s StepScreen) Render() string {
	var b strings.Builder

	title := fmt.Sprintf("%s - %s", strings.TrimSpace(s.Mode), strings.TrimSpace(s.StepName))
	writeSection(&b, "Step", title)
	writeSection(&b, "Description", strings.TrimSpace(s.Description))

	content := "(no content)"
	if len(s.Content) > 0 {
		content = strings.Join(s.Content, "\n")
	}
	writeSection(&b, "Content", content)

	if strings.TrimSpace(s.InlineError) != "" {
		writeSection(&b, "Error", s.InlineError)
	}

	writeSection(&b, "Actions", renderActions(s.Actions))
	return b.String()
}

func writeSection(b *strings.Builder, heading, body string) {
	border := strings.Repeat("=", len(heading)+8)
	b.WriteString(border)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("== %s ==\n", heading))
	b.WriteString(border)
	b.WriteString("\n")
	b.WriteString(body)
	b.WriteString("\n\n")
}

func renderActions(actions []Action) string {
	if len(actions) == 0 {
		return "(none)"
	}
	var rendered []string
	for _, a := range actions {
		label := fmt.Sprintf("[%s] %s", a.Key, a.Label)
		if a.Selected {
			label = fmt.Sprintf("> %s", label)
		}
		rendered = append(rendered, label)
	}
	return strings.Join(rendered, "   ")
}

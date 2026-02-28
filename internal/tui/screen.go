package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"
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
	return s.RenderWithWidth(72)
}

func (s StepScreen) RenderWithWidth(width int) string {
	var b strings.Builder
	if width < 40 {
		width = 40
	}
	ruleWidth := width - 2

	title := fmt.Sprintf("[%s] %s", strings.TrimSpace(s.Mode), strings.TrimSpace(s.StepName))
	desc := strings.TrimSpace(s.Description)
	if desc == "" {
		desc = "(no description)"
	}
	content := s.Content
	if len(content) == 0 {
		content = []string{"(no content)"}
	}

	writeRule(&b, ruleWidth)
	for _, line := range wrapToWidth(title, ruleWidth) {
		b.WriteString(line + "\n")
	}
	for _, line := range wrapToWidth(desc, ruleWidth) {
		b.WriteString(line + "\n")
	}
	writeRule(&b, ruleWidth)
	for _, line := range content {
		for _, wrapped := range wrapToWidth(line, ruleWidth) {
			b.WriteString(wrapped + "\n")
		}
	}
	if strings.TrimSpace(s.InlineError) != "" {
		writeRule(&b, ruleWidth)
		for _, line := range wrapToWidth("Error: "+strings.TrimSpace(s.InlineError), ruleWidth) {
			b.WriteString(line + "\n")
		}
	}
	writeRule(&b, ruleWidth)
	for _, line := range wrapToWidth(renderActions(s.Actions), ruleWidth) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	return b.String()
}

func writeRule(b *strings.Builder, width int) {
	b.WriteString(strings.Repeat("─", width))
	b.WriteString("\n")
}

func renderActions(actions []Action) string {
	if len(actions) == 0 {
		return "(none)"
	}
	var rendered []string
	for _, a := range actions {
		label := fmt.Sprintf("%s:%s", a.Key, a.Label)
		if a.Selected {
			label = fmt.Sprintf("[%s]", label)
		}
		rendered = append(rendered, label)
	}
	return "Actions: " + strings.Join(rendered, " | ")
}

func wrapToWidth(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	s = strings.TrimRight(s, " ")
	if s == "" {
		return []string{""}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var out []string
	cur := words[0]
	for _, w := range words[1:] {
		candidate := cur + " " + w
		if runeWidth(candidate) <= width {
			cur = candidate
			continue
		}
		out = append(out, cur)
		if runeWidth(w) > width {
			out = append(out, hardWrap(w, width)...)
			cur = ""
			continue
		}
		cur = w
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func hardWrap(s string, width int) []string {
	var out []string
	for runeWidth(s) > width {
		runes := []rune(s)
		out = append(out, string(runes[:width]))
		s = string(runes[width:])
	}
	if s != "" {
		out = append(out, s)
	}
	return out
}

func runeWidth(s string) int {
	return utf8.RuneCountInString(s)
}

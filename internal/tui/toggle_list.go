package tui

import "fmt"

type ToggleItem struct {
	ID             string
	Label          string
	Details        []string
	Enabled        bool
	DisabledReason string
}

func (i ToggleItem) Selectable() bool {
	return i.DisabledReason == ""
}

type ToggleList struct {
	Items  []ToggleItem
	Cursor int
}

func NewToggleList(items []ToggleItem) ToggleList {
	l := ToggleList{Items: items}
	l.Cursor = l.firstSelectableIndex()
	if l.Cursor < 0 {
		l.Cursor = 0
	}
	return l
}

func (l *ToggleList) MoveUp() {
	l.move(-1)
}

func (l *ToggleList) MoveDown() {
	l.move(1)
}

func (l *ToggleList) ToggleCurrent() (changed bool, reason string) {
	if len(l.Items) == 0 {
		return false, "no items to toggle"
	}
	item := &l.Items[l.Cursor]
	if !item.Selectable() {
		return false, item.DisabledReason
	}
	item.Enabled = !item.Enabled
	return true, ""
}

func (l ToggleList) Current() (ToggleItem, bool) {
	if len(l.Items) == 0 || l.Cursor < 0 || l.Cursor >= len(l.Items) {
		return ToggleItem{}, false
	}
	return l.Items[l.Cursor], true
}

func (l ToggleList) RenderLines() []string {
	if len(l.Items) == 0 {
		return []string{"(no items)"}
	}

	var lines []string
	for i, item := range l.Items {
		cursor := " "
		if i == l.Cursor {
			cursor = ">"
		}

		status := "[ ]"
		if item.Enabled {
			status = "[x]"
		}
		line := fmt.Sprintf("%s %s %s", cursor, status, item.Label)
		if !item.Selectable() {
			line = fmt.Sprintf("%s (disabled: %s)", line, item.DisabledReason)
		}
		lines = append(lines, line)
		for _, d := range item.Details {
			lines = append(lines, fmt.Sprintf("    - %s", d))
		}
	}
	return lines
}

func (l *ToggleList) move(delta int) {
	if len(l.Items) == 0 {
		return
	}
	next := l.Cursor
	for range len(l.Items) {
		next += delta
		if next < 0 {
			next = len(l.Items) - 1
		}
		if next >= len(l.Items) {
			next = 0
		}
		if l.Items[next].Selectable() {
			l.Cursor = next
			return
		}
	}
}

func (l ToggleList) firstSelectableIndex() int {
	for i, item := range l.Items {
		if item.Selectable() {
			return i
		}
	}
	return -1
}

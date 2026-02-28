package tui

import (
	"strings"
	"testing"
)

func TestStepScreenRenderIncludesRequiredSections(t *testing.T) {
	screen := StepScreen{
		Mode:        "Profile",
		StepName:    "Step 1a: Profiling Options",
		Description: "Review effective options and sources.",
		Content: []string{
			"safe=true (source: default -> cli)",
		},
		Actions: []Action{
			{Key: "enter", Label: "Accept", Selected: true},
			{Key: "b", Label: "Back"},
			{Key: "c", Label: "Cancel"},
		},
	}

	out := screen.Render()
	for _, required := range []string{"[Profile] Step 1a: Profiling Options", "Review effective options and sources.", "safe=true (source: default -> cli)", "Actions:"} {
		if !strings.Contains(out, required) {
			t.Fatalf("render missing content: %s\n%s", required, out)
		}
	}
}

func TestToggleListBlockedItemCannotToggle(t *testing.T) {
	l := NewToggleList([]ToggleItem{
		{ID: "dep", Label: "POST /auth/login", Enabled: true, DisabledReason: "required by GET /orders"},
		{ID: "main", Label: "GET /orders", Enabled: true},
	})
	l.Cursor = 0

	changed, reason := l.ToggleCurrent()
	if changed {
		t.Fatalf("expected blocked item not to toggle")
	}
	if reason == "" {
		t.Fatalf("expected disabled reason")
	}
}

func TestToggleListMoveSkipsDisabledItems(t *testing.T) {
	l := NewToggleList([]ToggleItem{
		{ID: "a", Label: "A", DisabledReason: "blocked"},
		{ID: "b", Label: "B", Enabled: false},
		{ID: "c", Label: "C", DisabledReason: "blocked"},
	})

	if l.Cursor != 1 {
		t.Fatalf("expected initial cursor on first selectable item, got %d", l.Cursor)
	}

	l.MoveDown()
	if l.Cursor != 1 {
		t.Fatalf("expected cursor to remain on only selectable item, got %d", l.Cursor)
	}
}

func TestToggleListClearsDerivationDetailsOnToggle(t *testing.T) {
	l := NewToggleList([]ToggleItem{
		{
			ID:                   "opt",
			Label:                "Option",
			Enabled:              true,
			Details:              []string{"source chain: default -> env"},
			ClearDetailsOnToggle: true,
		},
	})

	changed, reason := l.ToggleCurrent()
	if !changed || reason != "" {
		t.Fatalf("expected toggle success")
	}
	if len(l.Items[0].Details) != 0 {
		t.Fatalf("expected details to be cleared after toggle")
	}
}

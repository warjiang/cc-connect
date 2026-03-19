package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseUserQuestions_ValidInput(t *testing.T) {
	input := map[string]any{
		"questions": []any{
			map[string]any{
				"question":    "Which database?",
				"header":      "Setup",
				"multiSelect": false,
				"options": []any{
					map[string]any{"label": "PostgreSQL", "description": "Production"},
					map[string]any{"label": "SQLite", "description": "Dev"},
				},
			},
		},
	}
	qs := parseUserQuestions(input)
	if len(qs) != 1 {
		t.Fatalf("expected 1 question, got %d", len(qs))
	}
	q := qs[0]
	if q.Question != "Which database?" {
		t.Errorf("question = %q", q.Question)
	}
	if q.Header != "Setup" {
		t.Errorf("header = %q", q.Header)
	}
	if q.MultiSelect {
		t.Error("expected multiSelect=false")
	}
	if len(q.Options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(q.Options))
	}
	if q.Options[0].Label != "PostgreSQL" {
		t.Errorf("option[0].label = %q", q.Options[0].Label)
	}
	if q.Options[1].Description != "Dev" {
		t.Errorf("option[1].description = %q", q.Options[1].Description)
	}
}

func TestParseUserQuestions_EmptyInput(t *testing.T) {
	qs := parseUserQuestions(map[string]any{})
	if len(qs) != 0 {
		t.Errorf("expected 0 questions, got %d", len(qs))
	}
}

func TestParseUserQuestions_NoQuestionText(t *testing.T) {
	input := map[string]any{
		"questions": []any{
			map[string]any{"header": "Setup"},
		},
	}
	qs := parseUserQuestions(input)
	if len(qs) != 0 {
		t.Errorf("expected 0 questions (no question text), got %d", len(qs))
	}
}

func TestParseUserQuestions_MultiSelect(t *testing.T) {
	input := map[string]any{
		"questions": []any{
			map[string]any{
				"question":    "Select features",
				"multiSelect": true,
				"options": []any{
					map[string]any{"label": "Auth"},
					map[string]any{"label": "Logging"},
				},
			},
		},
	}
	qs := parseUserQuestions(input)
	if len(qs) != 1 {
		t.Fatalf("expected 1 question, got %d", len(qs))
	}
	if !qs[0].MultiSelect {
		t.Error("expected multiSelect=true")
	}
}

func TestNormalizePermissionMode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// dontAsk aliases
		{"dontAsk", "dontAsk"},
		{"dontask", "dontAsk"},
		{"dont-ask", "dontAsk"},
		{"dont_ask", "dontAsk"},
		// bypassPermissions aliases
		{"bypassPermissions", "bypassPermissions"},
		{"yolo", "bypassPermissions"},
		// acceptEdits aliases
		{"acceptEdits", "acceptEdits"},
		{"edit", "acceptEdits"},
		// plan
		{"plan", "plan"},
		// default fallback
		{"", "default"},
		{"unknown", "default"},
	}
	for _, tt := range tests {
		got := normalizePermissionMode(tt.input)
		if got != tt.want {
			t.Errorf("normalizePermissionMode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSummarizeInput_AskUserQuestion(t *testing.T) {
	input := map[string]any{
		"questions": []any{
			map[string]any{
				"question": "Which framework?",
				"options": []any{
					map[string]any{"label": "React"},
					map[string]any{"label": "Vue"},
				},
			},
		},
	}
	result := summarizeInput("AskUserQuestion", input)
	if result == "" {
		t.Error("expected non-empty summary for AskUserQuestion")
	}
}

func TestParseOptionEnv_FromSettingsJSON(t *testing.T) {
	opts := map[string]any{
		"settings_json": `{
			"env": {
				"DISABLE_TELEMETRY": "1",
				"ANTHROPIC_BASE_URL": "https://example.com",
				"API_TIMEOUT_MS": "3000000"
			}
		}`,
	}

	env, err := parseOptionEnv(opts)
	if err != nil {
		t.Fatalf("parseOptionEnv returned error: %v", err)
	}

	got := strings.Join(env, "\n")
	for _, expect := range []string{
		"DISABLE_TELEMETRY=1",
		"ANTHROPIC_BASE_URL=https://example.com",
		"API_TIMEOUT_MS=3000000",
	} {
		if !strings.Contains(got, expect) {
			t.Fatalf("missing env %q in %v", expect, env)
		}
	}
}

func TestParseOptionEnv_FromSettingsFile(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "claude-settings.json")
	content := `{
		"env": {
			"ANTHROPIC_AUTH_TOKEN": "token-123",
			"ANTHROPIC_MODEL": "deepseek-v3.2"
		}
	}`
	if err := os.WriteFile(settingsPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write settings file: %v", err)
	}

	env, err := parseOptionEnv(map[string]any{"settings_file": settingsPath})
	if err != nil {
		t.Fatalf("parseOptionEnv returned error: %v", err)
	}
	got := strings.Join(env, "\n")
	if !strings.Contains(got, "ANTHROPIC_AUTH_TOKEN=token-123") {
		t.Fatalf("missing ANTHROPIC_AUTH_TOKEN in %v", env)
	}
	if !strings.Contains(got, "ANTHROPIC_MODEL=deepseek-v3.2") {
		t.Fatalf("missing ANTHROPIC_MODEL in %v", env)
	}
}

func TestParseOptionEnv_MergePriority(t *testing.T) {
	opts := map[string]any{
		"settings_json": `{
			"env": {
				"ANTHROPIC_MODEL": "from-settings-json",
				"API_TIMEOUT_MS": "1000"
			}
		}`,
		"settings": map[string]any{
			"env": map[string]any{
				"ANTHROPIC_MODEL": "from-settings-map",
			},
		},
		"env": map[string]any{
			"ANTHROPIC_MODEL": "from-options-env",
		},
	}

	env, err := parseOptionEnv(opts)
	if err != nil {
		t.Fatalf("parseOptionEnv returned error: %v", err)
	}
	got := strings.Join(env, "\n")
	if !strings.Contains(got, "ANTHROPIC_MODEL=from-options-env") {
		t.Fatalf("expect highest priority value in %v", env)
	}
	if !strings.Contains(got, "API_TIMEOUT_MS=1000") {
		t.Fatalf("expect API_TIMEOUT_MS inherited from settings_json in %v", env)
	}
}

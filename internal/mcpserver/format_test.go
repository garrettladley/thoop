package mcpserver

import (
	"strings"
	"testing"
)

func TestTextEnvelopeTruncatesWithHint(t *testing.T) {
	t.Parallel()

	items := []map[string]string{
		{"id": "1", "body": strings.Repeat("a", 100)},
		{"id": "2", "body": strings.Repeat("b", 100)},
		{"id": "3", "body": strings.Repeat("c", 100)},
	}

	out := textEnvelope(items, PageInput{MaxTokens: 30}, envelopeOptions{
		Source:   "test",
		ToolName: "list_cycles",
		NextArgs: map[string]any{"start_date": "2026-05-01", "end_date": "2026-05-02", "max_tokens": 30},
	})
	if !strings.Contains(out, "<is_truncated>true</is_truncated>") {
		t.Fatalf("expected truncation metadata:\n%s", out)
	}
	if !strings.Contains(out, "start_at=") {
		t.Fatalf("expected next-page hint:\n%s", out)
	}
	if !strings.Contains(out, "<next_call>list_cycles(") {
		t.Fatalf("expected next_call:\n%s", out)
	}
	if !strings.Contains(out, "<source>test</source>") {
		t.Fatalf("expected source metadata:\n%s", out)
	}
}

func TestTextEnvelopeUsesStartAt(t *testing.T) {
	t.Parallel()

	items := []map[string]string{
		{"id": "1"},
		{"id": "2"},
	}

	out := textEnvelope(items, PageInput{StartAt: 1, MaxTokens: 1000}, envelopeOptions{})
	if strings.Contains(out, "id: \"1\"") {
		t.Fatalf("unexpected first item:\n%s", out)
	}
	if !strings.Contains(out, "id: \"2\"") {
		t.Fatalf("missing second item:\n%s", out)
	}
}

func TestTextEnvelopeUsesYAMLBody(t *testing.T) {
	t.Parallel()

	out := textEnvelope([]map[string]string{{"id": "1"}}, PageInput{MaxTokens: 1000}, envelopeOptions{})
	if !strings.Contains(out, "<YAML_DATA>\n- id: \"1\"") {
		t.Fatalf("expected YAML body:\n%s", out)
	}
	if strings.Contains(out, `{"id"`) {
		t.Fatalf("body looks like JSON:\n%s", out)
	}
}

package mcpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	go_json "github.com/goccy/go-json"
)

func TestConnectClaudeDesktopMergesConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "claude_desktop_config.json")
	existing := `{"mcpServers":{"other":{"command":"other","args":["serve"]}},"theme":"dark"}`
	writeFile(t, path, existing)

	result, err := ConnectClaudeDesktop(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Target != "claude-desktop" {
		t.Fatalf("target = %q", result.Target)
	}

	// #nosec G304 -- path is created by this test under t.TempDir.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := go_json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	servers := cfg["mcpServers"].(map[string]any)
	if _, ok := servers["other"]; !ok {
		t.Fatal("existing server was not preserved")
	}
	thoop := servers["thoop"].(map[string]any)
	if got := thoop["command"]; got != "thoop" {
		t.Fatalf("command = %v", got)
	}
}

func TestUpsertCodexBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing string
		want     []string
		reject   []string
	}{
		{
			name: "replaces existing block and preserves siblings",
			existing: strings.Join([]string{
				`model = "gpt-5"`,
				"",
				"[mcp_servers.other]",
				`command = "other"`,
				`args = ["serve"]`,
				"",
				"[mcp_servers.thoop]",
				`command = "old"`,
				`args = ["old"]`,
				"",
				"[history]",
				`persistence = "save-all"`,
			}, "\n"),
			want: []string{
				"[mcp_servers.other]",
				"[history]",
				`args = ["mcp", "serve"]`,
			},
			reject: []string{`command = "old"`},
		},
		{
			name:     "creates block in empty file",
			existing: "",
			want:     []string{"[mcp_servers.thoop]", `command = "thoop"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updated := upsertCodexBlock(tt.existing)
			for _, want := range tt.want {
				if !strings.Contains(updated, want) {
					t.Fatalf("missing %q in:\n%s", want, updated)
				}
			}
			for _, reject := range tt.reject {
				if strings.Contains(updated, reject) {
					t.Fatalf("unexpected %q in:\n%s", reject, updated)
				}
			}
			if count := strings.Count(updated, "[mcp_servers.thoop]"); count != 1 {
				t.Fatalf("thoop block count = %d", count)
			}
		})
	}
}

func TestParseConnectTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    ConnectTarget
		wantErr bool
	}{
		{name: "claude aggregate", input: "claude", want: ConnectTargetClaude},
		{name: "trims input", input: " codex-cli ", want: ConnectTargetCodexCLI},
		{name: "rejects unknown", input: "cursor", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseConnectTarget(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("target = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConnectWebReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	_, err := Connect(t.Context(), ConnectTargetWeb)
	if err == nil {
		t.Fatal("expected web not implemented error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

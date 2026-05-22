package mcpserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	go_json "github.com/goccy/go-json"
)

type ConnectResult struct {
	Target  string
	Changed []string
	Message string
}

type ConnectTarget string

const (
	ConnectTargetClaude        ConnectTarget = "claude"
	ConnectTargetClaudeCode    ConnectTarget = "claude-code"
	ConnectTargetClaudeDesktop ConnectTarget = "claude-desktop"
	ConnectTargetCodex         ConnectTarget = "codex"
	ConnectTargetCodexCLI      ConnectTarget = "codex-cli"
	ConnectTargetCodexApp      ConnectTarget = "codex-app"
	ConnectTargetWeb           ConnectTarget = "web"
)

func ParseConnectTarget(value string) (ConnectTarget, error) {
	target := ConnectTarget(strings.TrimSpace(value))
	switch target {
	case ConnectTargetClaude,
		ConnectTargetClaudeCode,
		ConnectTargetClaudeDesktop,
		ConnectTargetCodex,
		ConnectTargetCodexCLI,
		ConnectTargetCodexApp,
		ConnectTargetWeb:
		return target, nil
	default:
		return "", fmt.Errorf("unknown MCP connect target %q", value)
	}
}

func (t ConnectTarget) String() string {
	return string(t)
}

func Connect(ctx context.Context, target ConnectTarget) ([]ConnectResult, error) {
	switch target {
	case ConnectTargetClaude:
		return connectMany(ctx, ConnectTargetClaudeCode, ConnectTargetClaudeDesktop)
	case ConnectTargetCodex:
		return connectMany(ctx, ConnectTargetCodexCLI, ConnectTargetCodexApp)
	case ConnectTargetClaudeCode:
		result, err := ConnectClaudeCode(ctx)
		return oneResult(result, err)
	case ConnectTargetClaudeDesktop:
		result, err := ConnectClaudeDesktop("")
		return oneResult(result, err)
	case ConnectTargetCodexCLI:
		result, err := ConnectCodex("")
		if result.Target == "" {
			result.Target = ConnectTargetCodexCLI.String()
		}
		return oneResult(result, err)
	case ConnectTargetCodexApp:
		result, err := ConnectCodex("")
		if result.Target == "" || result.Target == ConnectTargetCodex.String() {
			result.Target = ConnectTargetCodexApp.String()
		}
		return oneResult(result, err)
	case ConnectTargetWeb:
		return nil, fmt.Errorf("hosted web MCP is not implemented yet; local MCP keeps credentials in the OS keyring")
	default:
		return nil, fmt.Errorf("unknown MCP connect target %q", target)
	}
}

func connectMany(ctx context.Context, targets ...ConnectTarget) ([]ConnectResult, error) {
	results := make([]ConnectResult, 0, len(targets))
	for _, target := range targets {
		targetResults, err := Connect(ctx, target)
		if err != nil {
			return results, err
		}
		results = append(results, targetResults...)
	}
	return results, nil
}

func oneResult(result ConnectResult, err error) ([]ConnectResult, error) {
	if err != nil {
		return nil, err
	}
	return []ConnectResult{result}, nil
}

func ConnectClaudeCode(ctx context.Context) (ConnectResult, error) {
	cfg := stdioConfig()
	data, err := go_json.Marshal(cfg)
	if err != nil {
		return ConnectResult{}, fmt.Errorf("marshal Claude Code MCP config: %w", err)
	}

	// #nosec G204 -- command and subcommands are fixed; data is generated MCP config JSON.
	cmd := exec.CommandContext(ctx, "claude", "mcp", "add-json", "thoop", string(data))
	if output, err := cmd.CombinedOutput(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return ConnectResult{}, fmt.Errorf("claude CLI not found; install Claude Code or configure Claude Desktop with `thoop mcp connect claude-desktop`")
		}
		return ConnectResult{}, fmt.Errorf("run claude mcp add-json: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return ConnectResult{
		Target:  "claude-code",
		Changed: []string{"Claude Code user MCP config"},
		Message: "Added thoop MCP server to Claude Code. Verify with `claude mcp get thoop`.",
	}, nil
}

func ConnectClaudeDesktop(configPath string) (ConnectResult, error) {
	if configPath == "" {
		var err error
		configPath, err = claudeDesktopConfigPath()
		if err != nil {
			return ConnectResult{}, err
		}
	}

	var cfg map[string]any
	// #nosec G304 -- configPath is either the known Claude Desktop config path or an explicit test override.
	if data, err := os.ReadFile(configPath); err == nil && len(bytes.TrimSpace(data)) > 0 {
		if err := go_json.Unmarshal(data, &cfg); err != nil {
			return ConnectResult{}, fmt.Errorf("parse Claude Desktop config %s: %w", configPath, err)
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return ConnectResult{}, fmt.Errorf("read Claude Desktop config %s: %w", configPath, err)
	}
	if cfg == nil {
		cfg = make(map[string]any)
	}

	servers, _ := cfg["mcpServers"].(map[string]any)
	if servers == nil {
		servers = make(map[string]any)
		cfg["mcpServers"] = servers
	}
	servers["thoop"] = stdioConfig()

	data, err := go_json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return ConnectResult{}, fmt.Errorf("marshal Claude Desktop config: %w", err)
	}
	if err := writeConfigFile(configPath, append(data, '\n')); err != nil {
		return ConnectResult{}, err
	}

	return ConnectResult{
		Target:  "claude-desktop",
		Changed: []string{configPath},
		Message: "Added thoop MCP server to Claude Desktop. Restart Claude Desktop to load it.",
	}, nil
}

func ConnectCodex(configPath string) (ConnectResult, error) {
	if configPath == "" {
		var err error
		configPath, err = codexConfigPath()
		if err != nil {
			return ConnectResult{}, err
		}
	}

	var existing string
	// #nosec G304 -- configPath is either the known Codex config path or an explicit test override.
	if data, err := os.ReadFile(configPath); err == nil {
		existing = string(data)
	} else if !errors.Is(err, os.ErrNotExist) {
		return ConnectResult{}, fmt.Errorf("read Codex config %s: %w", configPath, err)
	}

	updated := upsertCodexBlock(existing)
	if err := writeConfigFile(configPath, []byte(updated)); err != nil {
		return ConnectResult{}, err
	}

	return ConnectResult{
		Target:  "codex",
		Changed: []string{configPath},
		Message: "Added thoop MCP server to Codex. Restart Codex or run `/mcp` to verify.",
	}, nil
}

func stdioConfig() map[string]any {
	return map[string]any{
		"type":    "stdio",
		"command": "thoop",
		"args":    []string{"mcp", "serve"},
	}
}

func upsertCodexBlock(existing string) string {
	const header = "[mcp_servers.thoop]"
	block := strings.Join([]string{
		header,
		`type = "stdio"`,
		`command = "thoop"`,
		`args = ["mcp", "serve"]`,
	}, "\n") + "\n"

	lines := strings.Split(existing, "\n")
	out := make([]string, 0, len(lines)+4)
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != header {
			out = append(out, lines[i])
			continue
		}
		i++
		for i < len(lines) {
			trimmed := strings.TrimSpace(lines[i])
			if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
				i--
				break
			}
			i++
		}
	}

	updated := strings.TrimRight(strings.Join(out, "\n"), "\n")
	if updated != "" {
		updated += "\n\n"
	}
	updated += block
	return updated
}

func claudeDesktopConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "Claude", "claude_desktop_config.json"), nil
		}
		return filepath.Join(home, "AppData", "Roaming", "Claude", "claude_desktop_config.json"), nil
	default:
		return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"), nil
	}
}

func codexConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

func writeConfigFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

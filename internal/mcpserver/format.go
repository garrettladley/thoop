package mcpserver

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultMaxTokens = 10_000

type PageInput struct {
	StartAt   int `json:"start_at,omitempty" jsonschema:"Zero-based item offset for pagination."`
	MaxTokens int `json:"max_tokens,omitempty" jsonschema:"Approximate maximum response tokens. Default 10000."`
}

type envelopeOptions struct {
	Source       string
	FromCache    *bool
	PartialCache bool
	LastSync     *time.Time
	ToolName     string
	NextArgs     map[string]any
}

func textEnvelope(items any, page PageInput, opts envelopeOptions) string {
	if page.StartAt < 0 {
		page.StartAt = 0
	}
	if page.MaxTokens <= 0 {
		page.MaxTokens = defaultMaxTokens
	}

	total := sliceLen(items)
	displayed, displayedCount, next := pageItems(items, page.StartAt)
	maxChars := page.MaxTokens * 4

	for displayedCount > 0 && len(envelope(displayed, total, displayedCount, page.StartAt, next, opts, false)) > maxChars {
		displayedCount--
		displayed = takeSlice(items, page.StartAt, displayedCount)
		next = page.StartAt + displayedCount
	}

	isTruncated := next < total
	return envelope(displayed, total, displayedCount, page.StartAt, next, opts, isTruncated)
}

func singleEnvelope(item any, opts envelopeOptions) string {
	return envelope(item, 1, 1, 0, 1, opts, false)
}

func envelope(data any, total, displayed, startAt, next int, opts envelopeOptions, truncated bool) string {
	var b strings.Builder
	b.WriteString("<METADATA>\n")
	fmt.Fprintf(&b, "  <is_truncated>%t</is_truncated>\n", truncated)
	if truncated {
		fmt.Fprintf(&b, "  <truncation_message>Response truncated. Call again with start_at=%d to get the next batch, and/or increase max_tokens.</truncation_message>\n", next)
	}
	fmt.Fprintf(&b, "  <displayed_items>%d</displayed_items>\n", displayed)
	fmt.Fprintf(&b, "  <count>%d</count>\n", total)
	fmt.Fprintf(&b, "  <start_at>%d</start_at>\n", startAt)
	if truncated {
		fmt.Fprintf(&b, "  <next_start_at>%d</next_start_at>\n", next)
		if opts.ToolName != "" {
			fmt.Fprintf(&b, "  <next_call>%s</next_call>\n", formatNextCall(opts.ToolName, opts.NextArgs, next))
		}
	}
	if opts.Source != "" {
		fmt.Fprintf(&b, "  <source>%s</source>\n", opts.Source)
	}
	if opts.FromCache != nil {
		fmt.Fprintf(&b, "  <from_cache>%t</from_cache>\n", *opts.FromCache)
	}
	if opts.PartialCache {
		b.WriteString("  <partial_cache>true</partial_cache>\n")
	}
	if opts.LastSync != nil {
		fmt.Fprintf(&b, "  <last_sync>%s</last_sync>\n", opts.LastSync.Format(time.RFC3339))
	}
	b.WriteString("</METADATA>\n")
	b.WriteString("<YAML_DATA>\n")
	b.WriteString(toYAMLish(data))
	b.WriteString("\n</YAML_DATA>")
	return b.String()
}

func toYAMLish(v any) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Sprintf("error: %q", err.Error())
	}
	return strings.TrimSpace(string(data))
}

func formatNextCall(toolName string, args map[string]any, nextStartAt int) string {
	copied := make(map[string]any, len(args)+1)
	for key, value := range args {
		if value != "" && value != nil {
			copied[key] = value
		}
	}
	copied["start_at"] = nextStartAt

	keys := make([]string, 0, len(copied))
	for key := range copied {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, copied[key]))
	}
	return fmt.Sprintf("%s(%s)", toolName, strings.Join(parts, ", "))
}

func pageItems(items any, start int) (any, int, int) {
	total := sliceLen(items)
	if start >= total {
		return takeSlice(items, total, 0), 0, total
	}
	displayed := total - start
	return takeSlice(items, start, displayed), displayed, total
}

func sliceLen(items any) int {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return 1
	}
	return v.Len()
}

func takeSlice(items any, start, count int) any {
	v := reflect.ValueOf(items)
	if v.Kind() != reflect.Slice {
		return items
	}
	if start > v.Len() {
		start = v.Len()
	}
	end := min(start+count, v.Len())
	return v.Slice(start, end).Interface()
}

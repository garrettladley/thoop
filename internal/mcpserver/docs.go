package mcpserver

import (
	"fmt"
	"sort"
	"strings"

	_ "embed"
)

//go:embed docs/tools/list_thoop_skills.md
var toolListThoopSkills string

//go:embed docs/tools/load_thoop_skill.md
var toolLoadThoopSkill string

//go:embed docs/tools/get_latest_metrics.md
var toolGetLatestMetrics string

//go:embed docs/tools/get_cycle.md
var toolGetCycle string

//go:embed docs/tools/get_recovery.md
var toolGetRecovery string

//go:embed docs/tools/get_sleep.md
var toolGetSleep string

//go:embed docs/tools/list_cycles.md
var toolListCycles string

//go:embed docs/tools/list_recoveries.md
var toolListRecoveries string

//go:embed docs/tools/list_sleeps.md
var toolListSleeps string

//go:embed docs/tools/list_workouts.md
var toolListWorkouts string

//go:embed docs/skills/overview.md
var skillOverview string

//go:embed docs/skills/metrics.md
var skillMetrics string

//go:embed docs/skills/local-data.md
var skillLocalData string

//go:embed docs/skills/syncing.md
var skillSyncing string

//go:embed docs/skills/privacy.md
var skillPrivacy string

type Skill struct {
	Name        string
	Description string
	Related     []string
	Content     string
}

var toolDescriptions = map[string]string{
	"list_thoop_skills":  toolListThoopSkills,
	"load_thoop_skill":   toolLoadThoopSkill,
	"get_latest_metrics": toolGetLatestMetrics,
	"get_cycle":          toolGetCycle,
	"get_recovery":       toolGetRecovery,
	"get_sleep":          toolGetSleep,
	"list_cycles":        toolListCycles,
	"list_recoveries":    toolListRecoveries,
	"list_sleeps":        toolListSleeps,
	"list_workouts":      toolListWorkouts,
}

var skillDocs = map[string]string{
	"thoop/overview":   skillOverview,
	"thoop/metrics":    skillMetrics,
	"thoop/local-data": skillLocalData,
	"thoop/syncing":    skillSyncing,
	"thoop/privacy":    skillPrivacy,
}

func ToolDescription(name string) string {
	description, ok := toolDescriptions[name]
	if !ok {
		panic(fmt.Sprintf("missing embedded MCP tool description %q", name))
	}
	return strings.TrimSpace(description)
}

func ListSkills() ([]Skill, error) {
	names := make([]string, 0, len(skillDocs))
	for name := range skillDocs {
		names = append(names, name)
	}
	sort.Strings(names)

	skills := make([]Skill, 0, len(names))
	for _, name := range names {
		skill, err := LoadSkill(name)
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

func LoadSkill(name string) (Skill, error) {
	data, ok := skillDocs[name]
	if !ok {
		return Skill{}, fmt.Errorf("unknown thoop skill %q", name)
	}
	return parseSkill(data, name), nil
}

func parseSkill(data string, fallbackName string) Skill {
	content := strings.TrimSpace(data)
	skill := Skill{Name: fallbackName, Content: content}

	if !strings.HasPrefix(content, "---\n") {
		return skill
	}

	rest := strings.TrimPrefix(content, "---\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return skill
	}

	header := rest[:end]
	body := strings.TrimSpace(rest[end+len("\n---"):])
	skill.Content = body

	for _, line := range strings.Split(header, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "name":
			skill.Name = value
		case "description":
			skill.Description = value
		case "related":
			if value != "" {
				for _, related := range strings.Split(value, ",") {
					skill.Related = append(skill.Related, strings.TrimSpace(related))
				}
				sort.Strings(skill.Related)
			}
		}
	}

	return skill
}

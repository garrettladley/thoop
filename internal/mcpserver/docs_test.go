package mcpserver

import (
	"slices"
	"strings"
	"testing"
)

func TestEmbeddedToolDescriptions(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"list_thoop_skills",
		"load_thoop_skill",
		"get_latest_metrics",
		"get_cycle",
		"get_recovery",
		"get_sleep",
		"list_cycles",
		"list_recoveries",
		"list_sleeps",
		"list_workouts",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if desc := ToolDescription(name); strings.TrimSpace(desc) == "" {
				t.Fatalf("empty description for %s", name)
			}
		})
	}
}

func TestListAndLoadSkills(t *testing.T) {
	t.Parallel()

	skills, err := ListSkills()
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) == 0 {
		t.Fatal("expected embedded skills")
	}

	for _, listed := range skills {
		t.Run(listed.Name, func(t *testing.T) {
			t.Parallel()

			loaded, err := LoadSkill(listed.Name)
			if err != nil {
				t.Fatalf("load listed skill %s: %v", listed.Name, err)
			}
			if loaded.Description == "" {
				t.Fatalf("skill %s has empty description", listed.Name)
			}
			if loaded.Content == "" {
				t.Fatalf("skill %s has empty content", listed.Name)
			}
		})
	}
}

func TestListSkillsStableOrder(t *testing.T) {
	t.Parallel()

	skills, err := ListSkills()
	if err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		names = append(names, skill.Name)
		if !slices.IsSorted(skill.Related) {
			t.Fatalf("related skills for %s are not sorted: %v", skill.Name, skill.Related)
		}
	}
	if !slices.IsSorted(names) {
		t.Fatalf("skills are not sorted: %v", names)
	}
}

func TestLoadUnknownSkill(t *testing.T) {
	t.Parallel()

	if _, err := LoadSkill("thoop/nope"); err == nil {
		t.Fatal("expected unknown skill error")
	}
}

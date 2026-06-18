package skillcontent

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestListReadsSkillFrontmatter(t *testing.T) {
	reader := New(fstest.MapFS{
		"findo/SKILL.md": {Data: []byte("---\nname: findo\ndescription: >-\n  Chinese internet research\n---\nbody\n")},
	})

	skills, err := reader.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Fatalf("len(skills) = %d, want 1", len(skills))
	}
	if skills[0].Name != "findo" || skills[0].Description != "Chinese internet research" {
		t.Fatalf("unexpected skill info: %#v", skills[0])
	}
}

func TestReadSkillMarkdown(t *testing.T) {
	reader := New(fstest.MapFS{
		"findo/SKILL.md": {Data: []byte("---\nname: findo\n---\nbody\n")},
	})

	result, err := reader.Read("findo", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Skill != "findo" || result.Path != "SKILL.md" {
		t.Fatalf("unexpected result metadata: %#v", result)
	}
	if !strings.Contains(result.Content, "body") {
		t.Fatalf("content = %q", result.Content)
	}
	if !strings.Contains(result.Guidance, "findo skills read findo --json") {
		t.Fatalf("guidance = %q", result.Guidance)
	}
}

func TestReadRejectsUnknownSkill(t *testing.T) {
	reader := New(fstest.MapFS{
		"findo/SKILL.md": {Data: []byte("body\n")},
	})

	_, err := reader.Read("missing", "")
	if err == nil || !strings.Contains(err.Error(), "unknown skill") {
		t.Fatalf("error = %v", err)
	}
}

func TestReadRejectsTraversal(t *testing.T) {
	reader := New(fstest.MapFS{
		"findo/SKILL.md": {Data: []byte("body\n")},
	})

	tests := []string{"../x", "../../etc/passwd", `..\x`, "/tmp/x"}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			_, err := reader.Read("findo", tt)
			if err == nil || !strings.Contains(err.Error(), "invalid path") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

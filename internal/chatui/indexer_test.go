package chatui

import "testing"

func TestIndexEntryExported(t *testing.T) {
	e := IndexEntry{Name: "test", Kind: "skill", Source: "global"}
	if e.Name != "test" {
		t.Fatalf("expected Name=test, got %s", e.Name)
	}
}

func TestExtractDescription_Frontmatter(t *testing.T) {
	content := "---\nname: foo\ndescription: Does something useful\n---\n\n# Foo\n"
	got := extractDescription(content)
	want := "Does something useful"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractDescription_FallbackLine(t *testing.T) {
	content := "# My Skill\n\nThis skill does things.\n"
	got := extractDescription(content)
	want := "This skill does things."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractDescription_Empty(t *testing.T) {
	got := extractDescription("# Title\n\n")
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestExtractDescription_EmptyFrontmatterDesc(t *testing.T) {
	// Empty description: field falls through to body line
	content := "---\ndescription:\n---\n\nActual content here.\n"
	got := extractDescription(content)
	want := "Actual content here."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

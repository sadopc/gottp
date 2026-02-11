package templates

import (
	"testing"
)

func TestAll(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatal("expected templates, got none")
	}

	// Check all templates have required fields
	for _, tmpl := range all {
		if tmpl.Name == "" {
			t.Error("template has empty name")
		}
		if tmpl.Description == "" {
			t.Errorf("template %q has empty description", tmpl.Name)
		}
		if tmpl.Category == "" {
			t.Errorf("template %q has empty category", tmpl.Name)
		}
		if tmpl.Request == nil {
			t.Errorf("template %q has nil request", tmpl.Name)
		}
		if tmpl.Request != nil && tmpl.Request.URL == "" {
			t.Errorf("template %q has empty URL", tmpl.Name)
		}
	}
}

func TestCategories(t *testing.T) {
	cats := Categories()
	if len(cats) == 0 {
		t.Fatal("expected categories")
	}

	expected := map[string]bool{"REST": true, "GraphQL": true, "Auth": true, "WebSocket": true}
	for _, c := range cats {
		if !expected[c] {
			t.Errorf("unexpected category %q", c)
		}
	}
}

func TestByCategory(t *testing.T) {
	rest := ByCategory("REST")
	if len(rest) == 0 {
		t.Error("expected REST templates")
	}
	for _, tmpl := range rest {
		if tmpl.Category != "REST" {
			t.Errorf("expected REST category, got %q", tmpl.Category)
		}
	}

	none := ByCategory("NonExistent")
	if len(none) != 0 {
		t.Errorf("expected 0 templates for non-existent category, got %d", len(none))
	}
}

func TestByName(t *testing.T) {
	tmpl := ByName("GET JSON API")
	if tmpl == nil {
		t.Fatal("expected to find 'GET JSON API' template")
	}
	if tmpl.Request.Method != "GET" {
		t.Errorf("expected GET method, got %s", tmpl.Request.Method)
	}

	missing := ByName("NonExistent")
	if missing != nil {
		t.Error("expected nil for non-existent template")
	}
}

func TestTemplateCompleteness(t *testing.T) {
	// Ensure each category has at least one template
	for _, cat := range Categories() {
		templates := ByCategory(cat)
		if len(templates) == 0 {
			t.Errorf("category %q has no templates", cat)
		}
	}
}

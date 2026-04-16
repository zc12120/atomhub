package gemini

import "testing"

func TestParseModelsResponseParsesModels(t *testing.T) {
	body := []byte(`{"models":[{"name":"models/gemini-1.5-pro"},{"name":"models/gemini-1.5-flash"}]}`)
	models, err := ParseModels(body)
	if err != nil {
		t.Fatalf("parse models: %v", err)
	}
	if len(models) != 2 || models[0] != "gemini-1.5-pro" || models[1] != "gemini-1.5-flash" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestParseModelsResponseSkipsInvalidNames(t *testing.T) {
	body := []byte(`{"models":[{"name":"models/gemini-1.5-pro"},{"name":""},{"name":"models/gemini-1.5-pro"}]}`)
	models, err := ParseModels(body)
	if err != nil {
		t.Fatalf("parse models: %v", err)
	}
	if len(models) != 1 || models[0] != "gemini-1.5-pro" {
		t.Fatalf("unexpected models: %#v", models)
	}
}

func TestParseModelsResponseRejectsInvalidJSON(t *testing.T) {
	if _, err := ParseModels([]byte(`{"models":`)); err == nil {
		t.Fatalf("expected parse error")
	}
}
